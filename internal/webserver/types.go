package webserver

import (
	"database/sql"
	"github.com/CosminMocanu97/dissertationBackend/internal/mail"
)

type Service struct {
	Database       *sql.DB
	JwtSecret      string
	MailingService mail.Mailer
}