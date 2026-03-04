package notifier

import (
	"testing"

	"api/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMailClient_SSLMode(t *testing.T) {
	config := models.MailerConfiguration{
		Host:     "smtp.example.com",
		Port:     465,
		Username: "user",
		Password: "pass",
		Sender:   "test@example.com",
		TLSMode:  models.TLSModeSSL,
	}

	client, err := newMailClient(config)

	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewMailClient_STARTTLSMode(t *testing.T) {
	config := models.MailerConfiguration{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		Sender:   "test@example.com",
		TLSMode:  models.TLSModeStartTLS,
	}

	client, err := newMailClient(config)

	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewMailClient_NoneMode(t *testing.T) {
	config := models.MailerConfiguration{
		Host:    "mailpit",
		Port:    1025,
		Sender:  "test@example.com",
		TLSMode: models.TLSModeNone,
	}

	client, err := newMailClient(config)

	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewMailClient_SkipVerifyTLS_SSL(t *testing.T) {
	config := models.MailerConfiguration{
		Host:          "smtp.example.com",
		Port:          465,
		Username:      "user",
		Password:      "pass",
		Sender:        "test@example.com",
		TLSMode:       models.TLSModeSSL,
		SkipVerifyTLS: true,
	}

	client, err := newMailClient(config)

	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewMailClient_SkipVerifyTLS_STARTTLS(t *testing.T) {
	config := models.MailerConfiguration{
		Host:          "smtp.example.com",
		Port:          587,
		Username:      "user",
		Password:      "pass",
		Sender:        "test@example.com",
		TLSMode:       models.TLSModeStartTLS,
		SkipVerifyTLS: true,
	}

	client, err := newMailClient(config)

	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewMailClient_NoAuthWhenUsernameEmpty(t *testing.T) {
	config := models.MailerConfiguration{
		Host:    "mailpit",
		Port:    1025,
		Sender:  "test@example.com",
		TLSMode: models.TLSModeNone,
	}

	client, err := newMailClient(config)

	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewMailClient_WithCredentials(t *testing.T) {
	config := models.MailerConfiguration{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		Sender:   "test@example.com",
		TLSMode:  models.TLSModeStartTLS,
	}

	client, err := newMailClient(config)

	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewMailClient_SkipVerifyTLS_IgnoredWhenNone(t *testing.T) {
	config := models.MailerConfiguration{
		Host:          "mailpit",
		Port:          1025,
		Sender:        "test@example.com",
		TLSMode:       models.TLSModeNone,
		SkipVerifyTLS: true,
	}

	client, err := newMailClient(config)

	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewMailClient_InvalidTLSMode(t *testing.T) {
	config := models.MailerConfiguration{
		Host:    "smtp.example.com",
		Port:    587,
		Sender:  "test@example.com",
		TLSMode: "invalid",
	}

	client, err := newMailClient(config)

	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "unsupported TLS mode")
}
