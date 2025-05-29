package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// Sender interface defines email sending capabilities
type Sender interface {
	SendEmail(to, subject, body string) error
}

// GMailSender implements Sender using Gmail SMTP
type GMailSender struct {
	from     string
	password string
	smtpHost string
	smtpPort string
}

// NewGMailSender creates a new Gmail email sender
func NewGMailSender(from, password string) *GMailSender {
	return &GMailSender{
		from:     "bathblood085@gmail.com",
		password: "mqctcihrkgazuucc", // App Password, I know hardcoded but its small project
		smtpHost: "smtp.gmail.com",
		smtpPort: "465",
	}
}

// SendEmail sends an email using Gmail SMTP
func (s *GMailSender) SendEmail(to, subject, body string) error {
	// Set up authentication
	auth := smtp.PlainAuth("", s.from, s.password, s.smtpHost)

	// Prepare message
	msg := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\nMIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n%s",
		s.from, to, subject, body)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         s.smtpHost,
	}

	// Connect to server
	conn, err := tls.Dial("tcp", s.smtpHost+":"+s.smtpPort, tlsconfig)
	if err != nil {
		return fmt.Errorf("failed to dial SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.smtpHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Auth
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	// Set sender and recipient
	if err = client.Mail(s.from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	recipients := strings.Split(to, ",")
	for _, addr := range recipients {
		if err = client.Rcpt(strings.TrimSpace(addr)); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", addr, err)
		}
	}

	// Send email body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return client.Quit()
}

// MockSender for testing
type MockSender struct{}

func (m *MockSender) SendEmail(to, subject, body string) error {
	fmt.Printf("Mock email sent to: %s\nSubject: %s\nBody: %s\n", to, subject, body)
	return nil
}
