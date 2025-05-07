package core

import (
	"api/internal/models"
	"bytes"
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
	"html/template"
)

type Mailer struct {
	dialer *gomail.Dialer
	sender string
}

// NewMailer initializes the mailer and checks the connection
func NewMailer(config models.MailerConfiguration) *Mailer {
	dialer := gomail.NewDialer(config.Host, config.Port, config.Username, config.Password)

	connection, err := dialer.Dial()
	if err != nil {
		zap.L().Error("Failed to connect to mailer", zap.Error(err))
	}

	_ = connection.Close()

	return &Mailer{dialer: dialer, sender: config.Sender}
}

// NotifyFromTemplate sends an email using a given template and data
func (m *Mailer) NotifyFromTemplate(to string, subject string, templateName string, data interface{}) error {
	tmpl, err := template.ParseFiles(fmt.Sprintf("./internal/mails/%s.html", templateName))
	if err != nil {
		return err
	}

	var body bytes.Buffer
	if err = tmpl.Execute(&body, data); err != nil {
		return err
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.sender)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body.String())

	if err = m.dialer.DialAndSend(msg); err != nil {
		return err
	}

	return nil
}
