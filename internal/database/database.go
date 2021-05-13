package database

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/CosminMocanu97/dissertationBackend/internal/utils"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"

	_ "github.com/lib/pq"
)

const (
	TEST_ENVIRONMENT       = "test"
	// database connection consts
	host = "db"
	port = 5432
	user = "postgres"
)

// CreateDbConnection creates a connection to the postgres container
// if it runs for test, we create outside the docker env so we need to change the host
func CreateDbConnection(dbName string) (*sql.DB, error) {
	utils.GetEnvVars()

	databasePassword := os.Getenv("DATABASE_PASSWORD")

	var psqlInfo string
	if dbName == TEST_ENVIRONMENT {
		psqlInfo = fmt.Sprintf("host=%s port=%d user=%s "+
			"password=%s dbname=%s sslmode=disable",
			"0.0.0.0", port, user, databasePassword, dbName)
	} else {
		psqlInfo = fmt.Sprintf("host=%s port=%d user=%s "+
			"password=%s dbname=%s sslmode=disable",
			host, port, user, databasePassword, dbName)
	}

	log.Info("The connection string is %s", psqlInfo)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Error(err.Error())
		return db, err
	}

	return db, nil
}

func InitiateDatabaseTables(db *sql.DB) {
	// create the db tables
	err := CreateUsersTable(db)
	if err != nil {
		log.Fatal("Error creating the users table: %s", err)
	}
}
