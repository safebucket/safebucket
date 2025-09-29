package helpers

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

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
