package mail

import (
	"fmt"
	"net/smtp"
	"time"
)

// SMTPConfig holds configuration for the SMTP server
type SMTPConfig struct {
	Host     string // e.g., "smtp.gmail.com"
	Port     int    // e.g., 587
	Username string
	Password string
	From     string // sender email address
	Expiry   time.Duration
}

// SMTPSender holds SMTP config and can send emails
type SMTPSender struct {
	config SMTPConfig
}

// NewSMTPSender creates a new SMTPSender
func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{config: cfg}
}

// SendEmail sends a plain text email
type EmailRequest struct {
	To      string
	Subject string
	Body    string
}

func (s *SMTPSender) SendEmail(req EmailRequest) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", req.To, req.Subject, req.Body))

	return smtp.SendMail(addr, auth, s.config.From, []string{req.To}, msg)
}
