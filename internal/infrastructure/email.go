package infrastructure

import (
	"fmt"

	"github.com/resend/resend-go/v2"
)

type EmailClient interface {
	SendEmail(to []string, subject, htmlContent string) error
}

type resendClient struct {
	client *resend.Client
}

func NewEmailClient(apiKey string) EmailClient {
	client := resend.NewClient(apiKey)
	return &resendClient{client: client}
}

func (r *resendClient) SendEmail(to []string, subject, htmlContent string) error {
	params := &resend.SendEmailRequest{
		From:    "onboarding@resend.dev", // Configure this in env if needed
		To:      to,
		Subject: subject,
		Html:    htmlContent,
	}

	_, err := r.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}
