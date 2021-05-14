package database

import (
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	"os"

	"github.com/CosminMocanu97/dissertationBackend/pkg/log"

	_ "github.com/lib/pq"
)

const (
	host = "db"
	port = 5432
	user = "postgres"
)

// CreateDbConnection creates a connection to the postgres container
// if it runs for test, we create outside the docker env so we need to change the host
func CreateDbConnection(dbName string) (*sql.DB, error) {
	GetEnvVars()
	databasePassword := os.Getenv("DATABASE_PASSWORD")

	var psqlInfo string

	psqlInfo = fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		"0.0.0.0", port, user, databasePassword, dbName)


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

func GetEnvVars() {
	err := godotenv.Load("password.env")
	if err != nil {
		log.Fatal("Error loading the .env file")
	}
}

// CreateUsersTable test func for the db
func CreateUsersTable(db *sql.DB) error {
	createUsersQuery :=
		"create table if not exists users (  id serial primary key,  email text not null, passHash text not null, " +
			"isActivated bool not null default false, phoneNumber text not null, activationToken text not null);"
	_, err := db.Query(createUsersQuery)
	if err != nil {
		log.Error("Error creating the users table: %s", err)
	}

	log.Info("Successfully created users table")

	return err
}