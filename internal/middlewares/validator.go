package middlewares

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	h "api/internal/helpers"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"golang.org/x/text/unicode/norm"
)

type BodyKey struct{}

// validateName validates file, folder, and bucket names for S3 storage
// Allowed: Unicode letters/numbers, spaces, and special chars: _ - ( ) .
// Blocked: Path separators (/ \), path traversal (..), control chars, % (URL encoding issues), brackets.
func validateName(fl validator.FieldLevel) bool {
	name := fl.Field().String()

	name = norm.NFC.String(name)

	if strings.TrimSpace(name) == "" {
		return false
	}

	if strings.Contains(name, "..") {
		return false
	}

	for _, r := range name {
		// Block control characters (0x00-0x1F, 0x7F-0x9F)
		if r < 0x20 || (r >= 0x7F && r <= 0x9F) {
			return false
		}

		if r == '/' || r == '\\' {
			return false
		}

		// Block % to prevent URL encoding confusion
		if r == '%' {
			return false
		}

		// Block characters that are problematic in URLs or filesystems
		// Allowed special chars: _ - ( ) . and space
		// Blocked: : * ? " < > | [ ] and other special chars
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) &&
			r != '_' && r != '-' && r != '(' && r != ')' &&
			r != '.' && r != ' ' {
			return false
		}
	}

	return utf8.ValidString(name)
}

// validatePath validates folder paths for S3 storage
// Allows forward slashes for path segments but blocks other problematic characters
// Allowed: Unicode letters/numbers, spaces, forward slash (/), and special chars: _ - ( ) .
// Blocked: Backslash (\), path traversal (..), control chars, % (URL encoding issues), brackets.
func validatePath(fl validator.FieldLevel) bool {
	pathStr := fl.Field().String()

	pathStr = norm.NFC.String(pathStr)

	// Path can be empty or just "/"
	if pathStr == "" || pathStr == "/" {
		return true
	}

	// Block path traversal
	if strings.Contains(pathStr, "..") {
		return false
	}

	// Split by / and validate each segment (allow empty segments for leading/trailing slashes)
	segments := strings.Split(pathStr, "/")
	for _, segment := range segments {
		// Empty segments are OK (from leading/trailing slashes or consecutive slashes)
		if segment == "" {
			continue
		}

		// Each segment should follow the same rules as validateName (except allowing /)
		for _, r := range segment {
			// Block control characters (0x00-0x1F, 0x7F-0x9F)
			if r < 0x20 || (r >= 0x7F && r <= 0x9F) {
				return false
			}

			if r == '\\' {
				return false
			}

			// Block % to prevent URL encoding confusion
			if r == '%' {
				return false
			}

			// Block characters that are problematic in URLs or filesystems
			// Allowed special chars: _ - ( ) . and space
			// Blocked: : * ? " < > | [ ] and other special chars
			if !unicode.IsLetter(r) && !unicode.IsNumber(r) &&
				r != '_' && r != '-' && r != '(' && r != ')' &&
				r != '.' && r != ' ' {
				return false
			}
		}
	}

	return utf8.ValidString(pathStr)
}

// validateS3KeyLength validates that the combined S3 object key length doesn't exceed 1024 bytes
// This is a struct-level validator for FileTransferBody
// S3 key format: buckets/{uuid}/{path}/{name}
// UUID is 36 chars, so we check: 8 (buckets/) + 36 (uuid) + 1 (/) + len(path) + 1 (/) + len(name) <= 1024.
func validateS3KeyLength(sl validator.StructLevel) {
	// Type assertion to access the struct fields
	// We need to use reflection since this is a generic validator
	val := sl.Current()

	pathField := val.FieldByName("Path")
	nameField := val.FieldByName("Name")

	if !pathField.IsValid() || !nameField.IsValid() {
		return // Not a FileTransferBody, skip validation
	}

	path := pathField.String()
	name := nameField.String()

	// Calculate the S3 key: buckets/{uuid}/{path}/{name}
	// UUID is always 36 characters
	// "buckets/" = 8 bytes
	// uuid = 36 bytes
	// "/" = 1 byte
	// path = variable bytes (UTF-8)
	// "/" = 1 byte (if path doesn't end with /)
	// name = variable bytes (UTF-8)

	const uuidLength = 36
	const prefix = "buckets/"

	totalBytes := len([]byte(prefix)) + uuidLength // "buckets/{uuid}/"
	totalBytes += len([]byte(path))                // path bytes

	totalBytes += len([]byte(name))

	const maxS3KeyLength = 1024
	if totalBytes > maxS3KeyLength {
		sl.ReportError(nameField.Interface(), "Name", "Name", "s3keylength", "")
	}
}

func Validate[T any](next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB limit

		data := new(T)
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			zap.L().Error("failed to decode body", zap.Error(err))
			h.RespondWithError(w, http.StatusBadRequest, []string{"BAD_REQUEST"})
			return
		}

		validate := validator.New()
		_ = validate.RegisterValidation("filename", validateName)
		_ = validate.RegisterValidation("bucketname", validateName)
		_ = validate.RegisterValidation("filepath", validatePath)
		validate.RegisterStructValidation(validateS3KeyLength, data)

		err = validate.Struct(data)
		if err != nil {
			var strErrors []string
			for _, err := range func() validator.ValidationErrors {
				var target validator.ValidationErrors
				_ = errors.As(err, &target)
				return target
			}() {
				strErrors = append(strErrors, err.Error())
			}
			h.RespondWithError(w, http.StatusBadRequest, strErrors)
			return
		}

		ctx := context.WithValue(r.Context(), BodyKey{}, *data)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
