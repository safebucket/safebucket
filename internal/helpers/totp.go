package helpers

import (
	"fmt"

	"api/internal/configuration"

	"github.com/pquerna/otp/totp"
)

// TOTPKey holds the generated TOTP key information.
type TOTPKey struct {
	Secret string // Base32-encoded secret
	URL    string // otpauth:// URL for QR code generation
}

// GenerateTOTPSecret creates a new TOTP secret for the given email.
// Returns the secret and a URL that can be used to generate a QR code.
func GenerateTOTPSecret(email string) (*TOTPKey, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      configuration.AppName,
		AccountName: email,
		SecretSize:  20,
	})
	if err != nil {
		return nil, err
	}

	return &TOTPKey{
		Secret: key.Secret(),
		URL:    key.URL(),
	}, nil
}

// GenerateTOTPSecretWithEmail creates a TOTP URL using an existing secret and email.
func GenerateTOTPSecretWithEmail(email string, secret string) (*TOTPKey, error) {
	url := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		configuration.AppName, email, secret, configuration.AppName)
	return &TOTPKey{
		Secret: secret,
		URL:    url,
	}, nil
}

// ValidateTOTPCode validates a 6-digit TOTP code against the given secret.
func ValidateTOTPCode(secret string, code string) bool {
	return totp.Validate(code, secret)
}
