package mail

import (
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"

	"github.com/bwmarrin/snowflake"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// SendGridMailer is a wrapper over SendGrid, that is used to send emails
// the quota is 50 free emails/day, than we pay as we go
type SendGridMailer struct {
	SendGridAPIKey    string
	DefaultRecipients []string
}

// Mailer provides a function to send a mail having a subject, a plain payload and a HTML payload to a list of recipients
type Mailer interface {
	SendEmail([]string, string, string, string, snowflake.ID) error
}

func NewMailerService(sendGridAPIKey string) Mailer {
	return &SendGridMailer{
		SendGridAPIKey:    sendGridAPIKey,
	}
}

// SendEmail is an abstraction over sendGrid, in case we'll use something different in the future
func (mailer *SendGridMailer) SendEmail(recipientsEmail []string, subject, plainTextContent, htmlContent string, id snowflake.ID) error {
	for _, recipientEmail := range recipientsEmail {
		err := mailer.sendGridDeliverEmail(recipientEmail, subject, plainTextContent, htmlContent, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (mailer *SendGridMailer) sendGridDeliverEmail(recipientMail, subject, plainTextContent, htmlContent string, id snowflake.ID) error {
	from := mail.NewEmail("Dissertation", "cosmin@gstechnologies.io")
	to := mail.NewEmail("Example User", recipientMail)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(mailer.SendGridAPIKey)
	response, err := client.Send(message)

	if err != nil {
		log.Error("UUID: %s; Error sending email to %s with subject %s, plainTextContent %s and htmlContent %s: %s",
			id, recipientMail, subject, plainTextContent, htmlContent, err)
		log.Error("UUID: %s; response status code: %d, response body: %s", id, response.StatusCode, response.Body)
	} else {
		log.Info("UUID: %s; Successfully sent the email to %s with subject %s, plainTextContent %s and htmlContent %s",
			id, recipientMail, subject, plainTextContent, htmlContent)
	}
	return err
}