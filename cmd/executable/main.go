package main

import (
	"bufio"
	"flag"
	"github.com/CosminMocanu97/dissertationBackend/internal/mail"
	"os"

	"github.com/CosminMocanu97/dissertationBackend/internal/database"
	"github.com/CosminMocanu97/dissertationBackend/internal/utils"
	"github.com/CosminMocanu97/dissertationBackend/internal/webserver"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
	"time"
)

const (
	TEST_ENVIRONMENT       = "test"
	STAGING_ENVIRONMENT    = "staging"
	PRODUCTION_ENVIRONMENT = "production"
)

// utility function to retrieve the default email recipients from the file
func getDefaultEmailRecipients() []string {
	var defaultRecipients []string
	file, err := os.Open("default_email_recipients.txt")
	if err != nil {
		log.Fatal("Error opening the default_email_recipients.txt file: %s", err)
	}

	fScanner := bufio.NewScanner(file)
	for fScanner.Scan() {
		email := fScanner.Text()
		defaultRecipients = append(defaultRecipients, email)
	}

	return defaultRecipients
}

// @title GSTechnologies Auth Service API
// @version 1.0
// @description This is an auth service.
// @termsOfService https://gstechnologies.io/

// @contact.name gstechnologies
// @contact.url https://gstechnologies.io/
// @contact.email contact@gstechnologies.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
// @query.collection.format multi

// @x-extension-openapi {"example": "value on a json format"}
func main() {
	// get environment (either test or production)
	env := flag.String("env", "test", "Environment where the container will be deployed")
	flag.Parse()
	log.Info("The env is %s", *env)

	if *env != TEST_ENVIRONMENT && *env != PRODUCTION_ENVIRONMENT && *env != STAGING_ENVIRONMENT {
		log.Fatal("Unknown environment: %s", *env)
	}

	utils.GetEnvVars()

	// retrieve env vars
	serviceVersion := os.Getenv("SERVICE_VERSION")
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

	// create the db tables
	err = database.CreateUsersTable(db)
	if err != nil {
		log.Error("Error creating the users table: %s", err)
	}

	defer db.Close()

	database.InitiateDatabaseTables(db)

	defaultEmailRecipients := getDefaultEmailRecipients()
	mailer := mail.NewMailerService(sendGridAPIKey, defaultEmailRecipients)

	service := webserver.Service{
		Version:        serviceVersion,
		Database:       db,
		JwtSecret:      jwtSecret,
		MailingService: mailer,
	}
	a := webserver.Api(&service)
	err = a.Run(":8080")
	if err != nil {
		log.Error("Error starting the web server: %s", err.Error())
	}
}
