package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/Firakef1/settle/backend/internal/shared/config"
)

type ErrorType string

const (
	TypeAuthentication ErrorType = "AUTHENTICATION"
	TypeSendFailure    ErrorType = "SEND_FAILURE"
	TypeTimeout        ErrorType = "TIMEOUT"
)

type EmailError struct {
	Type    ErrorType
	Message string
	Err     error
}

func (e *EmailError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *EmailError) Unwrap() error {
	return e.Err
}

type Email struct {
	To      string
	Subject string
	Body    string
}

type EmailSender interface {
	Send(ctx context.Context, email Email) error
}

type SMTPEmailSender struct {
	config  config.EmailConfig
	logOnly bool
}

func NewSMTPSender(cfg config.EmailConfig) EmailSender {
	return &SMTPEmailSender{
		config:  cfg,
		logOnly: cfg.LogOnly,
	}
}

// NewEmailService constructs an EmailSender from global AppConfig.
func NewEmailService() EmailSender {
	return NewSMTPSender(config.AppConfig.Email)
}

func (s *SMTPEmailSender) Send(ctx context.Context, email Email) error {
	if s.logOnly {
		logTransportWarnOnce()
		fmt.Printf("[EMAIL LOG] To=%s Subject=%s\nBody:\n%s\n", email.To, email.Subject, email.Body)
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(email.To) == "" {
		return &EmailError{Type: TypeSendFailure, Message: "recipient address is required"}
	}
	if strings.TrimSpace(email.Subject) == "" {
		return &EmailError{Type: TypeSendFailure, Message: "subject is required"}
	}
	if strings.TrimSpace(email.Body) == "" {
		return &EmailError{Type: TypeSendFailure, Message: "body is required"}
	}
	if s.config.SMTPHost == "" {
		return &EmailError{Type: TypeAuthentication, Message: "smtp host is required"}
	}
	if s.config.SMTPUsername == "" {
		return &EmailError{Type: TypeAuthentication, Message: "smtp username is required"}
	}
	if strings.TrimSpace(s.config.SMTPPassword) == "" {
		return &EmailError{Type: TypeAuthentication, Message: "smtp password is required"}
	}

	fromAddr := strings.TrimSpace(s.config.FromAddress)
	if fromAddr == "" {
		fromAddr = strings.TrimSpace(s.config.SMTPUsername)
	}

	fromName := strings.TrimSpace(s.config.FromName)
	if fromName == "" {
		fromName = "Settle"
	}

	fromHeader := fmt.Sprintf("%s <%s>", fromName, fromAddr)

	timeout := s.config.SendTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	deadlineCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		deadlineCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	port := s.config.SMTPPort
	if port == "" {
		port = "587"
	}
	addr := net.JoinHostPort(s.config.SMTPHost, port)
	auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)

	msg := []byte("From: " + fromHeader + "\r\n" +
		"To: " + email.To + "\r\n" +
		"Subject: " + email.Subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		email.Body + "\r\n")

	if err := sendWithContext(deadlineCtx, s.config.SMTPHost, port, addr, auth, fromAddr, []string{email.To}, msg); err != nil {
		sanitizedErr := sanitizeSMTPError(err, s.config.SMTPPassword)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return &EmailError{Type: TypeTimeout, Message: "smtp send timed out or was cancelled", Err: sanitizedErr}
		}
		if isAuthError(err) {
			return &EmailError{Type: TypeAuthentication, Message: "smtp authentication failed", Err: sanitizedErr}
		}
		return &EmailError{Type: TypeSendFailure, Message: "smtp send failed", Err: sanitizedErr}
	}

	// fmt.Printf("[EMAIL SENT] To=%s From=%s\n", email.To, fromHeader)
	return nil
}

func sendWithContext(ctx context.Context, host, port, addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}

	type sendResult struct {
		err error
	}

	resultCh := make(chan sendResult, 1)
	go func() {
		if port == "465" {
			// Implicit TLS (SMTPS) for port 465
			resultCh <- sendResult{err: sendMailTLS(host, addr, auth, from, to, msg)}
		} else {
			// Plain / STARTTLS for port 587 or 25
			resultCh <- sendResult{err: smtp.SendMail(addr, auth, from, to, msg)}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-resultCh:
		return result.err
	}
}

func sendMailTLS(host, addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}

	for _, addr := range to {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("smtp rcpt to: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}

	return client.Quit()
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "535") || strings.Contains(msg, "auth") || strings.Contains(msg, "authentication")
}

func sanitizeSMTPError(err error, secret string) error {
	if err == nil {
		return nil
	}
	if secret == "" {
		return err
	}
	msg := err.Error()
	masked := MaskPassword(secret)
	return errors.New(strings.ReplaceAll(msg, secret, masked))
}

func MaskPassword(secret string) string {
	if len(secret) <= 4 {
		return "****"
	}
	return secret[:2] + "****" + secret[len(secret)-2:]
}
