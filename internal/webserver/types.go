package webserver

import (
	"database/sql"
)

type Service struct {
	Version        string
	Database       *sql.DB
	JwtSecret      string
}