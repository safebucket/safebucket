package notifier

import (
	"crypto/tls"
	"fmt"
	"html/template"

	"api/internal/models"

	mail "github.com/wneessen/go-mail"
	"go.uber.org/zap"
)

// SMTPNotifier implements INotifier using SMTP protocol.
type SMTPNotifier struct {
	client *mail.Client
	sender string
}

// NewSMTPNotifier initializes the SMTP notifier and checks the connection.
func NewSMTPNotifier(config models.MailerConfiguration) *SMTPNotifier {
	client, err := newMailClient(config)
	if err != nil {
		zap.L().Fatal("Failed to create SMTP client", zap.Error(err))
	}

	return &SMTPNotifier{client: client, sender: config.Sender}
}

func newMailClient(config models.MailerConfiguration) (*mail.Client, error) {
	opts := []mail.Option{
		mail.WithPort(config.Port),
	}

	switch config.TLSMode {
	case models.TLSModeSSL:
		opts = append(opts, mail.WithSSL())
	case models.TLSModeStartTLS:
		opts = append(opts, mail.WithTLSPolicy(mail.TLSMandatory))
	case models.TLSModeNone:
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	default:
		return nil, fmt.Errorf("unsupported TLS mode: %s", config.TLSMode)
	}

	if config.Username != "" {
		opts = append(opts,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(config.Username),
			mail.WithPassword(config.Password),
		)
	}

	client, err := mail.NewClient(config.Host, opts...)
	if err != nil {
		return nil, err
	}

	if config.TLSMode != models.TLSModeNone {
		// #nosec G402 -- InsecureSkipVerify is configurable for development environments.
		tlsConfig := &tls.Config{
			MinVersion:         tls.VersionTLS12,
			ServerName:         config.Host,
			InsecureSkipVerify: config.SkipVerifyTLS,
		}
		if err = client.SetTLSConfig(tlsConfig); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// NotifyFromTemplate sends an email using a given template and data.
func (s *SMTPNotifier) NotifyFromTemplate(
	to string,
	subject string,
	templateName string,
	data any,
) error {
	tmpl, err := template.ParseFiles(
		"./internal/mails/base.html",
		fmt.Sprintf("./internal/mails/%s.html", templateName),
	)
	if err != nil {
		return err
	}

	msg := mail.NewMsg()
	if err = msg.From(s.sender); err != nil {
		return err
	}
	if err = msg.To(to); err != nil {
		return err
	}
	msg.Subject(subject)
	if err = msg.SetBodyHTMLTemplate(tmpl, data); err != nil {
		return err
	}

	return s.client.DialAndSend(msg)
}
