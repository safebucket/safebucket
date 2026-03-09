package helpers

import (
	"fmt"

	"github.com/safebucket/safebucket/internal/configuration"

	"github.com/pquerna/otp/totp"
)

type TOTPKey struct {
	Secret string
	URL    string
}

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

func GenerateTOTPSecretWithEmail(email string, secret string) (*TOTPKey, error) {
	url := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		configuration.AppName, email, secret, configuration.AppName)
	return &TOTPKey{
		Secret: secret,
		URL:    url,
	}, nil
}

func ValidateTOTPCode(secret string, code string) bool {
	return totp.Validate(code, secret)
}
