package email

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/resend/resend-go/v3"
)

type EmailSender interface {
	SendVerificationCode(to, code, appName string) error
}

type resendClient interface {
	SendWithContext(ctx context.Context, params *resend.SendEmailRequest) (*resend.SendEmailResponse, error)
}

type resendSender struct {
	client resendClient
	from   string
}

func NewResendSender(apiKey string) EmailSender {
	return NewResendSenderWithFrom(apiKey, "noreply@spazio.com")
}

func NewResendSenderWithFrom(apiKey, from string) EmailSender {
	client := resend.NewClient(apiKey)
	return newResendSenderWithClient(client.Emails, from)
}

func newResendSenderWithClient(client resendClient, from string) EmailSender {
	return &resendSender{
		client: client,
		from:   strings.TrimSpace(from),
	}
}

func (s *resendSender) SendVerificationCode(to, code, appName string) error {
	to = strings.TrimSpace(to)
	code = strings.TrimSpace(code)
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "Spazio"
	}

	if s.client == nil {
		return errors.New("email client is not configured")
	}
	if s.from == "" {
		return errors.New("email sender address is not configured")
	}
	if to == "" || code == "" {
		return errors.New("email recipient and code are required")
	}

	escapedAppName := html.EscapeString(appName)
	escapedCode := html.EscapeString(code)
	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: fmt.Sprintf("Verifica tu cuenta en %s", appName),
		Html: fmt.Sprintf(
			`<p>Tu codigo de verificacion para %s es:</p><p style="font-size:24px;font-weight:700;letter-spacing:4px;">%s</p><p>Este codigo expira en 15 minutos.</p>`,
			escapedAppName,
			escapedCode,
		),
		Text: fmt.Sprintf("Tu codigo de verificacion para %s es: %s. Expira en 15 minutos.", appName, code),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := s.client.SendWithContext(ctx, params); err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}

	return nil
}
