package main

import (
	"os"
	"flag"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
	"github.com/CosminMocanu97/dissertationBackend/internal/mail"
	"github.com/CosminMocanu97/dissertationBackend/internal/database"
	"github.com/CosminMocanu97/dissertationBackend/internal/utils"
	"github.com/CosminMocanu97/dissertationBackend/internal/webserver"

	"time"
)

const (
	STAGING_ENVIRONMENT    = "staging"
)

func main() {
	// get environment (either test or production)
	env := flag.String("env", "staging", "Environment where the container will be deployed")
	flag.Parse()
	log.Info("The env is %s", *env)

	if *env != STAGING_ENVIRONMENT {
		log.Fatal("Unknown environment: %s", *env)
	}

	utils.GetEnvVars()

	// retrieve env vars
	jwtSecret := os.Getenv("JWT_SECRET")
	sendGridAPIKey := os.Getenv("SENDGRID_API_KEY")

	// sleep to give time to the Postgres container to start
	time.Sleep(time.Second * 5)

	db, err := database.CreateDbConnection(*env)
	if err != nil {
		log.Error("Error creating the database connection: %s", err.Error())
	} else {
		err = db.Ping()
		if err != nil {
			log.Fatal("Failed to ping db: %s", err.Error())
		}
		log.Info("Successfully created db connection")
	}

	defer db.Close()

	database.InitiateDatabaseTables(db)

	mailer := mail.NewMailerService(sendGridAPIKey)

	service := webserver.Service{
		Database:       db,
		JwtSecret: jwtSecret,
		MailingService: mailer,
	}
	a := webserver.Api(&service)
	err = a.Run(":8080")
	if err != nil {
		log.Error("Error starting the web server: %s", err.Error())
	}
}
