package notifier

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"io/fs"
	"strings"

	"github.com/safebucket/safebucket/internal/mails"
	"github.com/safebucket/safebucket/internal/models"

	mail "github.com/wneessen/go-mail"
	"go.uber.org/zap"
)

type SMTPNotifier struct {
	client    *mail.Client
	sender    string
	templates map[string]*template.Template
}

func NewSMTPNotifier(config models.MailerConfiguration) *SMTPNotifier {
	client, err := newMailClient(config)
	if err != nil {
		zap.L().Fatal("Failed to create SMTP client", zap.Error(err))
	}

	templates := parseMailTemplates()

	return &SMTPNotifier{client: client, sender: config.Sender, templates: templates}
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

func parseMailTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template)

	entries, err := fs.ReadDir(mails.TemplatesFS, ".")
	if err != nil {
		zap.L().Fatal("failed to read embedded mail templates", zap.Error(err))
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "base.html" || !strings.HasSuffix(name, ".html") {
			continue
		}

		tmpl, parseErr := template.ParseFS(mails.TemplatesFS, "base.html", name)
		if parseErr != nil {
			zap.L().Fatal("failed to parse mail template", zap.String("template", name), zap.Error(parseErr))
		}

		key := strings.TrimSuffix(name, ".html")
		templates[key] = tmpl
	}

	return templates
}

func (s *SMTPNotifier) NotifyFromTemplate(
	to string,
	subject string,
	templateName string,
	data any,
) error {
	if strings.ContainsAny(templateName, "/\\.") {
		return fmt.Errorf("invalid template name: %s", templateName)
	}

	tmpl, ok := s.templates[templateName]
	if !ok {
		return fmt.Errorf("unknown mail template: %s", templateName)
	}

	msg := mail.NewMsg()
	if err := msg.From(s.sender); err != nil {
		return err
	}
	if err := msg.To(to); err != nil {
		return err
	}
	msg.Subject(subject)
	if err := msg.SetBodyHTMLTemplate(tmpl, data); err != nil {
		return err
	}

	return s.client.DialAndSend(msg)
}
