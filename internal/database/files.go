package database

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
)

var extensions = []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx"}

type FilesDetails struct {
	ID   int64
	Name string
}

type SingleFileDetails struct {
	Filename   string
	Filepath   string
	OwnerID    int64
	RootFolder string
}

func CreateFilesTable(db *sql.DB) error {
	createFilesQuery :=
		"create table if not exists files (id serial primary key, ownerId bigint not null, folderId bigint not null, filename text not null, " +
			"filepath text not null);"
	_, err := db.Query(createFilesQuery)
	if err != nil {
		log.Error("Error creating the files table: %s", err)
	}

	log.Info("Successfully created files table")
	return err
}

func FileExists(db *sql.DB, folderID int64, name string) (bool, error) {
	fileExistsQuery :=
		"SELECT * FROM files WHERE folderid=$1 AND filename=$2;"
	res, err := db.Exec(fileExistsQuery, folderID, name)
	if err != nil {
		log.Error("Error checking if the files with name %s exists in folder with ID %d: %s", name, folderID, err)
		return false, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("Error retrieving the number of rows matching the query %s, for name %s: %s", fileExistsQuery, name, err)
		return false, err
	}

	if rowsAffected > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

/* func DoesFileExist(db *sql.DB, fileID int64, folderID int64, ownerID int64) (bool, error) {
	fileExistQuery :=
		"SELECT * FROM files WHERE id=$1 AND folderid=$2 AND ownerid=$3;"
	res, err := db.Exec(fileExistQuery, fileID, folderID, ownerID)
	if err != nil {
		log.Error("Error checking if the file with id %d exists: %s", fileID, err)
		return false, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("Error retrieving the number of rows matching the query %s, for ID %d: %s", fileExistQuery, fileID, err)
		return false, err
	}

	if rowsAffected > 0 {
		return true, nil
	} else {
		return false, nil
	}
}
*/
func checkExtension(slice []string, str string) bool {
	for _, value := range slice {
		if value == str {
			return true
		}
	}
	return false
}

func AddNewFile(db *sql.DB, userID int64, folderID int64, filename string, path string) (int64, error) {
	fileExtension := filepath.Ext(filename)
	//check if the extension is supported
	isFileExtensionValid := checkExtension(extensions, fileExtension)

	if !isFileExtensionValid {
		log.Error("Wrong extension file")
		return 0, errors.New("this extension is not supported")
	}

	fileAlreadyExists, gsErr := FileExists(db, folderID, filename)
	if gsErr != nil {
		log.Error("Error while checking if the file with the name %s exists in the folder with id %d", filename, folderID)
		return 0, gsErr
	}
	if fileAlreadyExists {
		log.Error("File %s already exists in the folder with ID %d", filename, folderID)
		return 0, errors.New("the specific file already exists in folder")
	}

	var fileID int64
	createNewFile := "INSERT INTO files(ownerid, folderid, filename, filepath) VALUES($1, $2, $3, $4) RETURNING id;"
	err := db.QueryRow(createNewFile, userID, folderID, filename, path).Scan(&fileID)
	if err != nil {
		log.Error("Error adding the file: %s into the file database: %s", filename, err)
		return 0, err
	}
	return fileID, nil
}

func UpdateFile(db *sql.DB, userID int64, folderID int64, filename string) (int64, error) {
	fileExtension := filepath.Ext(filename)
	//check if the extension is supported
	isFileExtensionValid := checkExtension(extensions, fileExtension)

	if !isFileExtensionValid {
		log.Error("Wrong extension file")
		return 0, errors.New("this extension is not supported")
	}

	var fileID int64
	updateExistingFile := "UPDATE files SET (ownerid, folderid, filename) = ($1, $2, $3) RETURNING id;"
	err := db.QueryRow(updateExistingFile, userID, folderID, filename).Scan(&fileID)
	if err != nil {
		log.Error("Error updating the file: %s into the database: %s", filename, err)
		return 0, err
	}
	return fileID, nil
}

func GetAllFilesDetails(db *sql.DB, folderID int64) ([]FilesDetails, error) {
	getAllFilesDetailsForFolderQuery :=
		"SELECT id, filename FROM files WHERE folderid=$1"
	rows, err := db.Query(getAllFilesDetailsForFolderQuery, folderID)
	if err != nil {
		log.Error("Error getting all files for folder with id %d: %s", folderID, err)
	}
	defer rows.Close()

	var allFilesDetails []FilesDetails
	for rows.Next() {
		var fileID int64
		var name string

		err = rows.Scan(&fileID, &name)
		if err != nil {
			log.Error("Error binding the files details for allFilesDetails request: %s", err)
			return allFilesDetails, err
		}
		filesDetails := new(FilesDetails)
		filesDetails.ID = fileID
		filesDetails.Name = name
		allFilesDetails = append(allFilesDetails, *filesDetails)
	}

	return allFilesDetails, nil
}

func GetFilesDetailsForFileID(db *sql.DB, fileID int64, folderID int64) (SingleFileDetails, error) {
	folderName, gsErr := GetFolderNameFromID(db, folderID)
	if gsErr != nil {
		log.Error("Error retriving the folderName for folderID %d", folderID)
		return SingleFileDetails{}, gsErr
	}

	filename, gsErr := GetFilenameFromID(db, fileID, folderID)
	if gsErr != nil {
		log.Error("Error retriving the filename for fileID %d", fileID)
		return SingleFileDetails{}, gsErr
	}

	doesFolderExist, err := FolderExists(db, folderName)
	if err != nil {
		log.Error("Error while checking if the folder already exists in db: %s", doesFolderExist)
		return SingleFileDetails{}, err
	}
	if !doesFolderExist {
		log.Error("There's no folder with the folderID %d", folderID)
		return SingleFileDetails{}, errors.New("folder doesnt exist")
	}

	doesFileExist, gsErr := FileExists(db, folderID, filename)
	if gsErr != nil {
		log.Error("Error while checking if the file with the name %s exists in the folder %s", filename, folderName)
		return SingleFileDetails{}, gsErr
	}
	if !doesFileExist {
		log.Error("There's no file in the folder %s", folderName)
		return SingleFileDetails{}, errors.New("file doesnt exist in folder")
	}

	getFilesDetailsForFileID :=
		"SELECT ownerid, filename, filepath FROM files WHERE id=$1 AND folderid=$2"
	rows, err := db.Query(getFilesDetailsForFileID, fileID, folderID)
	if err != nil {
		log.Error("Error getting the file name and path for id %d: %s", fileID, err)
	}
	defer rows.Close()

	var allFilesDetails SingleFileDetails
	for rows.Next() {
		var filename string
		var filepath string
		var ownerid int64

		err = rows.Scan(&ownerid, &filename, &filepath)
		if err != nil {
			log.Error("Error binding the files details for allFilesDetails request: %s", err)
			return allFilesDetails, err
		}
		allFilesDetails.Filename = filename
		allFilesDetails.Filepath = filepath
		allFilesDetails.OwnerID = ownerid
		allFilesDetails.RootFolder = folderName
	}

	return allFilesDetails, nil
}

func GetFilenameFromID(db *sql.DB, fileID int64, folderID int64) (string, error) {
	getFilenameForID := "SELECT filename FROM files WHERE id=$1"
	var filename string
	row := db.QueryRow(getFilenameForID, fileID)
	switch err := row.Scan(&filename); err {
	case sql.ErrNoRows:
		errorMessage := fmt.Sprintf("No file found for fileID %d", fileID)
		log.Error(errorMessage)
		return "", errors.New(errorMessage)
	case nil:
		log.Info("Successfully retrieved the name of the file with ID %d", fileID)
		return filename, nil
	default:
		log.Error("Error binding the filename for fileID %d: %s", fileID, err)
		return "", err
	}
}

func RemoveFile(db *sql.DB, fileID int64, folderID int64, ownerID int64) bool {
	deleteFileStatement := "DELETE FROM files WHERE id=$1 AND folderid=$2 AND ownerid=$3"
	res, err := db.Exec(deleteFileStatement, fileID, folderID, ownerID)
	if err != nil {
		log.Error("Error removing the file with ID %d from folder with ID: %s", fileID, folderID, err)
		return false
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("Error retrieving the number of rows affected from the query to delete the file with ID %d "+
			"from folder with the ID %d: %s", fileID, folderID, err)
		return false
	}
	if rowsAffected != 1 {
		errorMessage := fmt.Sprintf("There were %d rows affected, while there was 1 row expected", rowsAffected)
		log.Error("%s", errorMessage)
		return false
	}

	return true
}
