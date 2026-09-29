package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Firakef1/settle/backend/internal/shared/config"
)

func TestEmailService_Send_LogOnly(t *testing.T) {
	cfg := config.EmailConfig{LogOnly: true}
	sender := NewSMTPSender(cfg)

	err := sender.Send(context.Background(), Email{
		To:      "test@example.com",
		Subject: "Test Subject",
		Body:    "Test Body",
	})
	assert.NoError(t, err)
}

func TestEmailService_Send_ValidationErrors(t *testing.T) {
	cfg := config.EmailConfig{
		SMTPHost:     "localhost",
		SMTPPort:     "1025",
		SMTPUsername: "user",
		SMTPPassword: "password",
	}
	sender := NewSMTPSender(cfg)

	// Missing recipient
	err := sender.Send(context.Background(), Email{Subject: "S", Body: "B"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recipient address is required")

	// Missing subject
	err = sender.Send(context.Background(), Email{To: "a@b.com", Body: "B"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subject is required")

	// Missing body
	err = sender.Send(context.Background(), Email{To: "a@b.com", Subject: "S"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "body is required")
}

func TestEmailService_MaskPassword(t *testing.T) {
	assert.Equal(t, "****", MaskPassword("123"))
	assert.Equal(t, "se****et", MaskPassword("secret"))
}
