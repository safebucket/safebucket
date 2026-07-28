package storage

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureStorageAPIVersion = "2023-11-03"

const (
	azureHeaderDate      = "x-ms-date"
	azureHeaderVersion   = "x-ms-version"
	azureHeaderBlobType  = "x-ms-blob-type"
	azureBlobTypeBlock   = "BlockBlob"
	azureMetaHeaderPfx   = "x-ms-meta-"
	azureBlockIDDigits   = 32
	azureAuthHeaderName  = "Authorization"
	azureContentTypeName = "Content-Type"
)

type azureSharedKeySigner struct {
	accountName string
	key         []byte
}

func newAzureSharedKeySigner(accountName, accountKeyBase64 string) (*azureSharedKeySigner, error) {
	key, err := base64.StdEncoding.DecodeString(accountKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode azure account key: %w", err)
	}
	return &azureSharedKeySigner{accountName: accountName, key: key}, nil
}

type azureSignRequest struct {
	Method        string
	Container     string
	BlobPath      string
	Query         url.Values
	ContentLength int64
	ContentType   string
	XMSHeaders    map[string]string
}

func buildStringToSign(account string, req azureSignRequest) string {
	contentLength := ""
	if req.ContentLength > 0 {
		contentLength = strconv.FormatInt(req.ContentLength, 10)
	}

	lines := []string{
		req.Method,
		"",            // Content-Encoding
		"",            // Content-Language
		contentLength, // Content-Length
		"",            // Content-MD5
		req.ContentType,
		"",
		"",
		"",
		"",
		"",
		"", // Range
	}

	return strings.Join(lines, "\n") + "\n" +
		canonicalizedHeaders(req.XMSHeaders) +
		canonicalizedResource(account, req.Container, req.BlobPath, req.Query)
}

func canonicalizedHeaders(xmsHeaders map[string]string) string {
	lower := make(map[string]string, len(xmsHeaders))
	keys := make([]string, 0, len(xmsHeaders))
	for k, v := range xmsHeaders {
		lk := strings.ToLower(k)
		lower[lk] = v
		keys = append(keys, lk)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(lower[k])
		b.WriteByte('\n')
	}
	return b.String()
}

func canonicalizedResource(account, container, blobPath string, query url.Values) string {
	var b strings.Builder
	b.WriteByte('/')
	b.WriteString(account)
	b.WriteByte('/')
	b.WriteString(container)
	if blobPath != "" {
		b.WriteByte('/')
		b.WriteString(blobPath)
	}

	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		values := append([]string(nil), query[k]...)
		sort.Strings(values)
		b.WriteByte('\n')
		b.WriteString(strings.ToLower(k))
		b.WriteByte(':')
		b.WriteString(strings.Join(values, ","))
	}
	return b.String()
}

func (s *azureSharedKeySigner) sign(req azureSignRequest) string {
	stringToSign := buildStringToSign(s.accountName, req)

	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("SharedKey %s:%s", s.accountName, signature)
}

func (s *azureSharedKeySigner) headersForPut(
	container, blobPath string,
	query url.Values,
	contentLength int64,
	contentType string,
	extraHeaders map[string]string,
) map[string]string {
	xmsHeaders := make(map[string]string, len(extraHeaders)+2)
	for k, v := range extraHeaders {
		if strings.HasPrefix(strings.ToLower(k), "x-ms-") {
			xmsHeaders[k] = v
		}
	}
	xmsHeaders[azureHeaderDate] = time.Now().UTC().Format(http.TimeFormat)
	xmsHeaders[azureHeaderVersion] = azureStorageAPIVersion

	auth := s.sign(azureSignRequest{
		Method:        http.MethodPut,
		Container:     container,
		BlobPath:      blobPath,
		Query:         query,
		ContentLength: contentLength,
		ContentType:   contentType,
		XMSHeaders:    xmsHeaders,
	})

	headers := make(map[string]string, len(xmsHeaders)+2)
	for k, v := range xmsHeaders {
		headers[k] = v
	}
	headers[azureAuthHeaderName] = auth
	if contentType != "" {
		headers[azureContentTypeName] = contentType
	}
	return headers
}

func azureBlockID(partNumber int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%0*d", azureBlockIDDigits, partNumber)))
}

func azurePartNumberFromBlockID(blockID string) (int, error) {
	decoded, err := base64.StdEncoding.DecodeString(blockID)
	if err != nil {
		return 0, fmt.Errorf("decode block id %q: %w", blockID, err)
	}

	trimmed := strings.TrimLeft(string(decoded), "0")
	if trimmed == "" {
		trimmed = "0"
	}

	partNumber, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("parse part number from block id %q: %w", blockID, err)
	}
	return partNumber, nil
}
