package helpers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ValidateProviderName validates that a provider name is safe to use
// Max length: 50 characters
// Allowed characters: alphanumeric, underscore, hyphen.
func ValidateProviderName(providerName string) error {
	if len(providerName) == 0 {
		return errors.New("provider name cannot be empty")
	}
	if len(providerName) > 50 {
		return errors.New("provider name exceeds maximum length of 50 characters")
	}

	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, providerName)
	if err != nil {
		return err
	}
	if !matched {
		return errors.New("provider name contains invalid characters")
	}

	return nil
}

func RandString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func SetCallbackCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   int(time.Hour.Seconds()),
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
}

func IsDomainAllowed(email string, domains []string) bool {
	email = strings.TrimSpace(email)
	if err := validate.Var(email, "email"); err != nil {
		return false
	}

	if len(domains) == 0 {
		return true
	}

	emailDomain := strings.ToLower(strings.Split(email, "@")[1])

	for _, domain := range domains {
		domain = strings.TrimSpace(strings.ToLower(domain))
		if domain != "" && emailDomain == domain {
			return true
		}
	}

	return false
}
