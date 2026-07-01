package email

import (
	"bytes"
	"context"
	"embed"
	"text/template"

	"github.com/m-bromo/go-auth-template/configs"
	"github.com/resend/resend-go/v3"
)

type ResendClient struct {
	client        *resend.Client
	resendOptions *configs.Resend
}

//go:embed templates/send-otp-code.html
var templateFS embed.FS

const otpTemplatePath = "templates/send-otp-code.html"

func NewResendClient(resendOptions *configs.Resend) *ResendClient {
	client := resend.NewClient(resendOptions.ApiKey)

	return &ResendClient{
		client:        client,
		resendOptions: resendOptions,
	}
}

func (s *ResendClient) SendCode(ctx context.Context, email string, code string) error {
	tpml, err := template.ParseFS(templateFS, otpTemplatePath)
	if err != nil {
		return err
	}

	var htmlBuffer bytes.Buffer
	if err := tpml.Execute(&htmlBuffer, code); err != nil {
		return err
	}

	params := &resend.SendEmailRequest{
		From:    s.resendOptions.Email,
		To:      []string{email},
		Subject: "OTP code",
		Html:    htmlBuffer.String(),
	}

	if _, err := s.client.Emails.SendWithContext(ctx, params); err != nil {
		return err
	}

	return nil
}
