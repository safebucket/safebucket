package notifier

import (
	"testing"

	"api/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureDialer_SSLMode(t *testing.T) {
	config := models.MailerConfiguration{
		Host:     "smtp.example.com",
		Port:     465,
		Username: "user",
		Password: "pass",
		TLSMode:  models.TLSModeSSL,
	}

	dialer := configureDialer(config)

	assert.True(t, dialer.SSL)
	require.NotNil(t, dialer.TLSConfig)
	assert.Equal(t, "smtp.example.com", dialer.TLSConfig.ServerName)
	assert.False(t, dialer.TLSConfig.InsecureSkipVerify)
}

func TestConfigureDialer_STARTTLSMode(t *testing.T) {
	config := models.MailerConfiguration{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		TLSMode:  models.TLSModeStartTLS,
	}

	dialer := configureDialer(config)

	assert.False(t, dialer.SSL)
	require.NotNil(t, dialer.TLSConfig)
	assert.Equal(t, "smtp.example.com", dialer.TLSConfig.ServerName)
	assert.False(t, dialer.TLSConfig.InsecureSkipVerify)
}

func TestConfigureDialer_NoneMode(t *testing.T) {
	config := models.MailerConfiguration{
		Host:     "mailpit",
		Port:     1025,
		Username: "user",
		Password: "pass",
		TLSMode:  models.TLSModeNone,
	}

	dialer := configureDialer(config)

	assert.False(t, dialer.SSL)
	assert.Nil(t, dialer.TLSConfig)
}

func TestConfigureDialer_SkipVerifyTLS_SSL(t *testing.T) {
	config := models.MailerConfiguration{
		Host:          "smtp.example.com",
		Port:          465,
		Username:      "user",
		Password:      "pass",
		TLSMode:       "ssl",
		SkipVerifyTLS: true,
	}

	dialer := configureDialer(config)

	require.NotNil(t, dialer.TLSConfig)
	assert.True(t, dialer.TLSConfig.InsecureSkipVerify)
}

func TestConfigureDialer_SkipVerifyTLS_STARTTLS(t *testing.T) {
	config := models.MailerConfiguration{
		Host:          "smtp.example.com",
		Port:          587,
		Username:      "user",
		Password:      "pass",
		TLSMode:       "starttls",
		SkipVerifyTLS: true,
	}

	dialer := configureDialer(config)

	require.NotNil(t, dialer.TLSConfig)
	assert.True(t, dialer.TLSConfig.InsecureSkipVerify)
}

func TestConfigureDialer_AuthDisabledWhenUsernameEmpty(t *testing.T) {
	config := models.MailerConfiguration{
		Host:    "mailpit",
		Port:    1025,
		TLSMode: models.TLSModeNone,
	}

	dialer := configureDialer(config)

	assert.Nil(t, dialer.Auth)
}

func TestConfigureDialer_CredentialsPreservedWhenUsernameSet(t *testing.T) {
	config := models.MailerConfiguration{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		TLSMode:  models.TLSModeStartTLS,
	}

	dialer := configureDialer(config)

	assert.Equal(t, "user", dialer.Username)
	assert.Equal(t, "pass", dialer.Password)
}
