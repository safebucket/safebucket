package helpers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"math/big"
	"strings"

	"github.com/safebucket/safebucket/internal/models"
)

const (
	TokenPrefixPersonal       = "sb_pat_"
	TokenPrefixServiceAccount = "sb_sat_"

	tokenSecretLength   = 32
	tokenChecksumLength = 8
	tokenCharset        = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func TokenPrefixForProvider(providerType models.ProviderType) string {
	if providerType == models.ServiceAccountProviderType {
		return TokenPrefixServiceAccount
	}
	return TokenPrefixPersonal
}

func randomTokenSecret() (string, error) {
	b := make([]byte, tokenSecretLength)
	maxIndex := big.NewInt(int64(len(tokenCharset)))
	for i := range b {
		n, err := rand.Int(rand.Reader, maxIndex)
		if err != nil {
			return "", err
		}
		b[i] = tokenCharset[n.Int64()]
	}
	return string(b), nil
}

func tokenChecksum(secret string) string {
	return fmt.Sprintf("%0*x", tokenChecksumLength, crc32.ChecksumIEEE([]byte(secret)))
}

func hashTokenBody(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func GenerateAPIToken(providerType models.ProviderType) (string, string, error) {
	secret, err := randomTokenSecret()
	if err != nil {
		return "", "", err
	}
	body := secret + tokenChecksum(secret)
	return TokenPrefixForProvider(providerType) + body, hashTokenBody(body), nil
}

func HasAPITokenPrefix(token string) bool {
	return strings.HasPrefix(token, TokenPrefixPersonal) ||
		strings.HasPrefix(token, TokenPrefixServiceAccount)
}

func ParseAPIToken(token string) (string, error) {
	var body string
	switch {
	case strings.HasPrefix(token, TokenPrefixPersonal):
		body = strings.TrimPrefix(token, TokenPrefixPersonal)
	case strings.HasPrefix(token, TokenPrefixServiceAccount):
		body = strings.TrimPrefix(token, TokenPrefixServiceAccount)
	default:
		return "", errors.New("invalid token prefix")
	}

	if len(body) != tokenSecretLength+tokenChecksumLength {
		return "", errors.New("invalid token length")
	}

	secret := body[:tokenSecretLength]
	if body[tokenSecretLength:] != tokenChecksum(secret) {
		return "", errors.New("invalid token checksum")
	}

	return hashTokenBody(body), nil
}
