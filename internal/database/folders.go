package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
)

var (
	NAME_IS_EMPTY = "folder name cannot be empty"
	FOLDER_ALREADY_EXISTS = "the folder already exists in the database"
)

type FolderDetails struct {
	ID int64
	Name string
}

type SingleFolderDetails struct {
	Name string
	OwnerID int64
}

func CreateFoldersTable(db *sql.DB) error {
	createFilesQuery :=
		"CREATE TABLE if not exists folders (id serial primary key, ownerId bigint not null, name text not null);"
	_, err := db.Query(createFilesQuery)
	if err != nil {
		log.Error("Error creating the folders table: %s", err)
	}

	log.Info("Successfully created folders table")

	return err
}

func FolderExists(db *sql.DB, name string) (bool, error) {
	folderExistsQuery :=
		"SELECT * FROM folders WHERE name=$1;"
	res, err := db.Exec(folderExistsQuery, name)
	if err != nil {
		log.Error("Error checking if the folder with name %s exists: %s", name, err)
		return false, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("Error retrieving the number of rows matching the query %s, for name %s: %s", folderExistsQuery, name, err)
		return false, err
	}

	if rowsAffected > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func AddNewFolder(db *sql.DB, userID int64, folderName string) (int64, error) {
	folderAlreadyExists, err := FolderExists(db, folderName)
	if err != nil {
		log.Error("Error while checking if the folder %s already exists in the database", folderName)
		return 0, err
	}  else if len(folderName) == 0 {
		log.Error("The folder name is empty")
		return 0, errors.New(NAME_IS_EMPTY)
	}	else if folderAlreadyExists {
		log.Error("Folder %s already exists in the database", folderName)
		return 0, errors.New(FOLDER_ALREADY_EXISTS)
	} else {
		if !folderAlreadyExists {
			var folderID int64

			addNewFolderStatement :=
				"INSERT INTO folders(ownerId, name) VALUES($1, $2) RETURNING id;"

			err := db.QueryRow(addNewFolderStatement, userID, folderName).Scan(&folderID)
			if err != nil {
				log.Error("Error adding the new folder: ", err)
				return 0, err
			}

			log.Info("Successfully created the folder %s", folderName)
			return folderID, nil
		} else {
			err = errors.New(FOLDER_ALREADY_EXISTS)
			return 0, err
		}
	}
}

func GetAllFoldersDetails(db *sql.DB) ([]FolderDetails, error) {
	getAllFoldersDetailsQuery :=
		"SELECT id, name FROM folders"
	rows, err := db.Query(getAllFoldersDetailsQuery)
	if err != nil {
		log.Error("Error getting the data for all the folders: %s", err)
	}
	defer rows.Close()

	var allFolderDetails []FolderDetails
	for rows.Next() {
		var folderId int64
		var name string

		err = rows.Scan(&folderId, &name)
		if err != nil {
			log.Error("Error binding the data for GetAllFoldersDetails request: %s", err)
			return allFolderDetails, err
		}
		folderDetails := new(FolderDetails)
		folderDetails.ID = folderId
		folderDetails.Name = name
		allFolderDetails = append(allFolderDetails, *folderDetails)
	}
	return allFolderDetails, nil
}

func GetAllFoldersDetailsForID(db *sql.DB, folderID int64) (SingleFolderDetails, error) {
	getSingleFolderDetailsQuery :=
		"SELECT ownerid, name FROM folders WHERE id=$1"
	rows, err := db.Query(getSingleFolderDetailsQuery, folderID)
	if err != nil {
		log.Error("Error getting the folder details for the ID %d: %s",folderID, err)
	}
	defer rows.Close()

	var singleFolderDetails SingleFolderDetails
	for rows.Next() {
		var ownerid int64
		var name string

		err = rows.Scan(&ownerid, &name)
		if err != nil {
			log.Error("Error retrieving the details for the GetAllFoldersDetailsForID request : %s", err)
			return singleFolderDetails, err
		}
		
		singleFolderDetails.Name = name
		singleFolderDetails.OwnerID = ownerid
	}
	return singleFolderDetails, nil
}

func GetFolderNameFromID(db *sql.DB, folderID int64) (string, error) {
	getFolderNameForID := "SELECT name FROM folders WHERE id=$1"
	var folderName string
	row := db.QueryRow(getFolderNameForID, folderID)
	switch err := row.Scan(&folderName); err {
	case sql.ErrNoRows:
		errorMessage := fmt.Sprintf("No folder was found for ID %d", folderID)
		log.Error(errorMessage)
		return "", errors.New(errorMessage)
	case nil:
		log.Info("Successfully retrieved the name of the folder with ID %d", folderID)
		return folderName, nil
	default:
		log.Error("Error binding the folder name for folderID %d: %s", folderID, err)
		return "", err
	}
}

func RemoveFolder(db *sql.DB, folderID int64, ownerID int64) bool {
	deleteFolderStatement := "DELETE FROM folders WHERE id=$1 AND ownerid=$2"
	res, err := db.Exec(deleteFolderStatement, folderID, ownerID)
	if err != nil {
		log.Error("Error removing the folder with ID %d for user with ID: %s", folderID, ownerID, err)
		return false
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("Error retrieving the number of rows affected from the query to delete the folder with ID %d "+
			"for user with the ID %d: %s", folderID, ownerID, err)
		return false
	}
	if rowsAffected != 1 {
		errorMessage := fmt.Sprintf("There were %d rows affected, while there was 1 row expected: %s", rowsAffected, err)
		log.Error("%s", errorMessage)
		return false
	}

	return true
}

func RemoveSubfoldersFromFolder(db *sql.DB, folderID int64, userID int64) bool {
	deleteSubfoldersFromFolderStatement := "DELETE FROM subfolders WHERE folderid=$1 AND ownerid=$2"
	_, err := db.Exec(deleteSubfoldersFromFolderStatement, folderID, userID)
	if err != nil {
		log.Error("Error removing the subfolders from folder with ID %d: %s", folderID, err)
		return false
	}
	return true
}

func RemoveFilesFromFolder(db *sql.DB, folderID int64, userID int64) bool {
	deleteFilesFromFolderStatement := "DELETE FROM files WHERE folderid=$1 AND ownerid=$2"
	_, err := db.Exec(deleteFilesFromFolderStatement, folderID, userID)
	if err != nil {
		log.Error("Error removing the files from folder with ID %d: %s", folderID, err)
		return false
	}
	return true
}

