package database

import (
	"database/sql"
	"errors"
	"fmt"

	//"errors"
	//"fmt"
	//"github.com/bwmarrin/snowflake"

	//"github.com/CosminMocanu97/dissertationBackend/internal/auth"
	//"github.com/CosminMocanu97/dissertationBackend/pkg/gserror"
	//"github.com/CosminMocanu97/dissertationBackend/internal/types"
	"github.com/CosminMocanu97/dissertationBackend/internal/auth"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
)

var (
	SUBFOLDERNAME_IS_EMPTY = "subfolder name cannot be empty"
	SUBFOLDER_ALREADY_EXISTS = "this subfolder already exists in the database"
)

type SubfolderDetails struct {
	ID int64
	Name string
}

type SingleSubfolderDetails struct {
	Name string
	OwnerID int64
	Password string
	IsLocked bool
	RootFolder string
}

func CreateSubfoldersTable(db *sql.DB) error {
	createFilesQuery :=
		"CREATE TABLE if not exists subfolders (id serial primary key, ownerId bigint not null, folderId bigint not null, name text not null, " +
			"password text, isLocked bool not null );"
	_, err := db.Query(createFilesQuery)
	if err != nil {
		log.Error("Error creating the subfolders table: %s", err)
	}

	log.Info("Successfully created subfolders table")

	return err
}

func SubfolderExists(db *sql.DB, folderID int64, name string) (bool, error) {
	subfolderExistsQuery :=
		"SELECT * FROM subfolders WHERE folderid=$1 AND name=$2;"
	res, err := db.Exec(subfolderExistsQuery, folderID, name)
	if err != nil {
		log.Error("Error checking if the subfolder with name %s exists: %s", name, err)
		return false, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("Error retrieving the number of rows matching the query %s, for name %s: %s", subfolderExistsQuery, name, err)
		return false, err
	}

	if rowsAffected > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func AddNewSubfolder(db *sql.DB, userID int64, folderID int64, subfolderName string, password string, isLocked bool) (int64, error) {
	subfolderAlreadyExists, err := SubfolderExists(db, folderID, subfolderName)
	if err != nil {
		log.Error("Error while checking if the subfolder %s already exists in the database", subfolderName)
		return 0, err
	} else if len(subfolderName) == 0 {
		log.Error("The subfolder name cannot be empty")
		return 0, errors.New(SUBFOLDERNAME_IS_EMPTY)
	} else if subfolderAlreadyExists {
		log.Error("Folder %s already exists in the database", subfolderName)
		return 0, errors.New(SUBFOLDER_ALREADY_EXISTS)
	} else {
		// if the email is not already in the database, try to add it
		if !subfolderAlreadyExists {
			var subfolderID int64
			passHash := ""
			if len(password) > 0 {
				passHash = auth.ComputePasswordHash(password)
			}

			addNewFolderStatement :=
				"INSERT INTO subfolders(ownerId, folderId, name, password, isLocked) VALUES($1, $2, $3, $4, $5) RETURNING id;"

			err := db.QueryRow(addNewFolderStatement, userID, folderID, subfolderName, passHash, isLocked).Scan(&subfolderID)
			if err != nil {
				log.Error("Error adding the new subfolder: ", err)
				return 0, err
			}

			log.Info("Successfully created the subfolder %s", subfolderName)
			return subfolderID, nil
		} else {
			err = errors.New(SUBFOLDER_ALREADY_EXISTS)
			return 0, err
		}
	}
}

func GetAllSubFoldersDetails(db *sql.DB, folderID int64) ([]SubfolderDetails, error) {
	getAllSubfoldersDetailsQuery :=
		"SELECT id, name FROM subfolders where folderid=$1"
	rows, err := db.Query(getAllSubfoldersDetailsQuery, folderID)
	if err != nil {
		log.Error("Error getting the data for all the subfolders for folderID %d: %s", folderID, err)
	}
	defer rows.Close()

	var allSubfolderDetails []SubfolderDetails
	for rows.Next() {
		var subfolderID int64
		var name string

		err = rows.Scan(&subfolderID, &name)
		if err != nil {
			log.Error("Error binding the data for GetAllSubFoldersDetails request: %s", err)
			return allSubfolderDetails, err
		}
		subfolderDetails := new(SubfolderDetails)
		subfolderDetails.ID = subfolderID
		subfolderDetails.Name = name
		allSubfolderDetails = append(allSubfolderDetails, *subfolderDetails)
	}
	return allSubfolderDetails, nil
}

func GetAllSubfolderDetailsForID(db *sql.DB, subfolderID int64, folderID int64) (SingleSubfolderDetails, error) {
	folderName, gsErr := GetFolderNameFromID(db, folderID)
	if gsErr != nil {
		log.Error("Error retriving the folderName for folderID %d", folderID)
		return SingleSubfolderDetails{}, gsErr
	}
	subfolderName, gsErr := GetSubfolderNameFromID(db, subfolderID)
	if gsErr != nil {
		log.Error("Error retriving the name for subfolder with ID %d", subfolderID)
		return SingleSubfolderDetails{}, gsErr
	}

	doesFolderExist, err := FolderExists(db, folderName)
	if err != nil {
		log.Error("Error while checking if the folder already exists in db: %s", doesFolderExist)
		return SingleSubfolderDetails{}, err
	}
	if !doesFolderExist {
		log.Error("There's no folder with the folderID %d", folderID)
		return SingleSubfolderDetails{}, errors.New("folder doesnt exist")
	}

	doesSubfolderExist, err := SubfolderExists(db, folderID, subfolderName)
	if err != nil {
		log.Error("Error while checking if the subfolder already exists in db: %s", doesSubfolderExist)
		return SingleSubfolderDetails{}, err
	}
	if !doesSubfolderExist {
		log.Error("There's no subfolder with the ID %d", subfolderID)
		return SingleSubfolderDetails{}, errors.New("subfolder doesnt exist")
	}

	getSingleSubfolderDetailsQuery :=
		"SELECT ownerid, name, password, islocked FROM subfolders WHERE id=$1 AND folderid=$2"
	rows, err := db.Query(getSingleSubfolderDetailsQuery, subfolderID, folderID)
	if err != nil {
		log.Error("Error getting the subfolder details for the ID %d: %s", subfolderID, err)
	}
	defer rows.Close()

	var singleSubfolderDetails SingleSubfolderDetails
	for rows.Next() {
		var ownerid int64
		var name string
		var password string
		var isLocked bool

		err = rows.Scan(&ownerid, &name, &password, &isLocked)
		if err != nil {
			log.Error("Error retrieving the details for the GetAllSubfolderDetailsForID request : %s", err)
			return singleSubfolderDetails, err
		}

		singleSubfolderDetails.Name = name
		singleSubfolderDetails.OwnerID = ownerid
		singleSubfolderDetails.Password = password
		singleSubfolderDetails.IsLocked = isLocked
		singleSubfolderDetails.RootFolder = folderName
	}
	return singleSubfolderDetails, nil
}

func GetSubfolderNameFromID(db *sql.DB, subfolderID int64) (string, error) {
	getSubfolderNameForID := "SELECT name FROM subfolders WHERE id=$1"
	var subfolderName string
	row := db.QueryRow(getSubfolderNameForID, subfolderID)
	switch err := row.Scan(&subfolderName); err {
	case sql.ErrNoRows:
		errorMessage := fmt.Sprintf("No subfolder was found for ID %d", subfolderID)
		log.Error(errorMessage)
		return "", errors.New(errorMessage)
	case nil:
		log.Info("Successfully retrieved the name of the subfolder with ID %d", subfolderID)
		return subfolderName, nil
	default:
		log.Error("Error binding the subfolder name for subfolderID %d: %s", subfolderID, err)
		return "", err
	}
}

func VerifySubfolderPassword(db *sql.DB, subfolderID int64, folderID int64, password string) (bool, error) {
	currentSubfolderDetails, err := GetAllSubfolderDetailsForID(db, subfolderID, folderID)
	if err != nil {
		log.Error("Error retrieving the details for subfolderID %d; %s", folderID, err)
		return false, err
	}

	// verify if the password matches
	passHash := auth.ComputePasswordHash(password)
	if currentSubfolderDetails.Password == passHash {
		log.Info("The provided password for subfolder %s is correct", currentSubfolderDetails.Name)
		return true, nil
	} else {
		log.Info("The user tried to access the subfolder %s but the password was incorrect", currentSubfolderDetails.Name)
		return false, nil
	}
}

func RemoveSubfolder(db *sql.DB, subfolderID int64, folderID int64, ownerID int64) bool {
	deleteFolderStatement := "DELETE FROM subfolders WHERE id=$1 AND folderid=$2 AND ownerid=$3"
	res, err := db.Exec(deleteFolderStatement, subfolderID, folderID, ownerID)
	if err != nil {
		log.Error("Error removing the subfolder with ID %d from folder with ID: %s", subfolderID, folderID, err)
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

func RemoveFilesFromSubfolder(db *sql.DB, subfolderID int64, folderID int64, userID int64) bool {
	deleteFilesFromFolderStatement := "DELETE FROM files WHERE subfolderid=$1 AND folderid=$2 AND ownerid=$3"
	_, err := db.Exec(deleteFilesFromFolderStatement, subfolderID, folderID, userID)
	if err != nil {
		log.Error("Error removing the files from folder with ID %d: %s", folderID, err)
		return false
	}
	return true
}
