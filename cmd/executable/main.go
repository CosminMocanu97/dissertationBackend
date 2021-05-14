package main

import (
	_ "bufio"
	"flag"
	_ "os"

	"github.com/CosminMocanu97/dissertationBackend/internal/database"
	"github.com/CosminMocanu97/dissertationBackend/internal/webserver"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
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

	service := webserver.Service{
		Database:       db,
	}
	a := webserver.Api(&service)
	err = a.Run(":8080")
	if err != nil {
		log.Error("Error starting the web server: %s", err.Error())
	}
}
