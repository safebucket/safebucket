package notifier

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"

	"api/internal/models"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

// SMTPNotifier implements INotifier using SMTP protocol.
type SMTPNotifier struct {
	dialer *gomail.Dialer
	sender string
}

// NewSMTPNotifier initializes the SMTP notifier and checks the connection.
func NewSMTPNotifier(config models.MailerConfiguration) *SMTPNotifier {
	dialer := gomail.NewDialer(config.Host, config.Port, config.Username, config.Password)

	if config.EnableTLS {
		dialer.SSL = true
		// #nosec G402 -- InsecureSkipVerify is configurable for development environments
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: config.SkipVerifyTLS}
	} else {
		dialer.SSL = false
	}

	if config.Username == "" {
		dialer.Auth = nil
	}

	connection, err := dialer.Dial()
	if err != nil {
		zap.L().Error("Failed to connect to SMTP server", zap.Error(err))
	} else {
		_ = connection.Close()
	}

	return &SMTPNotifier{dialer: dialer, sender: config.Sender}
}

// NotifyFromTemplate sends an email using a given template and data.
func (s *SMTPNotifier) NotifyFromTemplate(
	to string,
	subject string,
	templateName string,
	data interface{},
) error {
	tmpl, err := template.ParseFiles(fmt.Sprintf("./internal/mails/%s.html", templateName))
	if err != nil {
		return err
	}

	var body bytes.Buffer
	if err = tmpl.Execute(&body, data); err != nil {
		return err
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", s.sender)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body.String())

	if err = s.dialer.DialAndSend(msg); err != nil {
		return err
	}

	return nil
}
