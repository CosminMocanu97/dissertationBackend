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

	var psqlInfo string = fmt.Sprintf("host=%s port=%d user=%s "+
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

	err = CreateSubfoldersTable(db)
	if err != nil {
		log.Fatal("Error creating the subfolders table: %s", err)
	}

	err = CreateFilesTable(db)
	if err != nil {
		log.Fatal("Error creating the files table: %s", err)
	}

	err = CreateFoldersTable(db)
	if err != nil {
		log.Fatal("Error creating the folders table: %s", err)
	}

}

func GetEnvVars() {
	err := godotenv.Load("password.env")
	if err != nil {
		log.Fatal("Error loading the .env file")
	}
}
