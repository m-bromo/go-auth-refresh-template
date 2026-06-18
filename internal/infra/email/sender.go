package email

import (
	"bytes"
	"context"
	"embed"
	"text/template"

	"github.com/m-bromo/go-auth-template/config"
	"github.com/resend/resend-go/v3"
)

type EmailSender interface {
	SendCode(ctx context.Context, email string, code string) error
}

type emailSender struct {
	client *resend.Client
	cfg    *config.Config
}

//go:embed templates/send-otp-code.html
var templateFS embed.FS

const otpTemplatePath = "templates/send-otp-code.html"

func NewEmailSender(cfg *config.Config) EmailSender {
	client := resend.NewClient(cfg.Resend.ApiKey)

	return &emailSender{
		client: client,
		cfg:    cfg,
	}
}

func (s *emailSender) SendCode(ctx context.Context, email string, code string) error {
	tpml, err := template.ParseFS(templateFS, otpTemplatePath)
	if err != nil {
		return err
	}

	var htmlBuffer bytes.Buffer
	if err := tpml.Execute(&htmlBuffer, code); err != nil {
		return err
	}

	params := &resend.SendEmailRequest{
		From:    s.cfg.Resend.Email,
		To:      []string{email},
		Subject: "OTP code",
		Html:    htmlBuffer.String(),
	}

	if _, err := s.client.Emails.Send(params); err != nil {
		return err
	}

	return nil
}
