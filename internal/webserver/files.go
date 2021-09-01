package webserver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/CosminMocanu97/dissertationBackend/internal/database"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
	"github.com/gin-gonic/gin"
)

var (
	pathToSaveFiles = "/home/cosminel/DissertationAppFolders/"
	invalidFileExtension = "this extension is not supported"
	fileAlreadyExists = "the specific file already exists in subfolder"
)

type VerifyFilePassword struct {
	Password string `json:"password"`
}

func (s *Service) HandlePostAddFile(c *gin.Context) {
	claims, err := verifyClaims(c)
	if err != nil {
		// if the claims not exist, mark it as unauthorised, otherwise, when the account is not activated,
		// just return, so the status code is 403, from the verifyClaims logic
		if err.Error() == ClaimsNotExist {
			log.Error("Error retrieving the claims from JWT")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": ClaimsNotExist,
			})
		}
		return
	}

	folderID, err := getIntParameterFromRequest(c, "folder_id")
	if err != nil {
		log.Error("Error retrieving folder_id parameter from the "+
			"getIntParameterFromRequest request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	subfolderID, err := getIntParameterFromRequest(c, "subfolder_id")
	if err != nil {
		log.Error("Error retrieving subfolder_id parameter from the "+
			"getIntParameterFromRequest request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		log.Error("Error getting the file from the form: %s", err.Error())
		c.Status(http.StatusBadRequest)
		return
	}

	folderName, err := database.GetFolderNameFromID(s.Database, folderID)
	if err != nil {
		log.Error("Error getting the folderName %s from the folderID %d: %s", folderName, folderID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	subfolderName, err := database.GetSubfolderNameFromID(s.Database, subfolderID)
	if err != nil {
		log.Error("Error getting the subfolderName %s from the subfolderID %d: %s", subfolderName, subfolderID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	var fileLocked bool
	password := c.Request.FormValue("password")
	if len(password) == 0 {
		fileLocked = false
	} else {
		fileLocked = true
	}

	fullPath := pathToSaveFiles + folderName + "/" + subfolderName + "/"

 	fileID, gsErr := database.AddNewFile(s.Database, claims.Id, folderID, subfolderID, file.Filename, fullPath, password, fileLocked)
	if gsErr != nil {
		errorMessage := fmt.Sprintf("Error saving the file %s: %s", file.Filename, gsErr)
		log.Error(errorMessage)

		if gsErr.Error() == fileAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{
				"error": errorMessage,
			})
			return
		} else if gsErr.Error() == invalidFileExtension {
			c.JSON(http.StatusForbidden, gin.H{
				"error": errorMessage,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": errorMessage,
		})
		return
	} else if err := c.SaveUploadedFile(file, fullPath + file.Filename); err != nil {
		errorMessage := fmt.Sprintf("Error while saving the file: %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errorMessage,
		})
		return
	}

	log.Info("File %s successfully uploaded!", file.Filename)
	c.JSON(http.StatusOK, gin.H{
		"id": fileID,
	})
}

func (s *Service) HandleGetAllFilesForCurrentFolder(c *gin.Context) {
	folderID, err := getIntParameterFromRequest(c, "folder_id")
	if err != nil {
		log.Error("Error retrieving folder_id parameter from the "+
			"HandleGetAllFilesForCurrentFolder request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}
	
	subfolderID, err := getIntParameterFromRequest(c, "subfolder_id")
	if err != nil {
		log.Error("Error retrieving subfolder_id parameter from the "+
			"getIntParameterFromRequest request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	folderName, err := database.GetFolderNameFromID(s.Database, folderID)
	if err != nil {
		log.Error("Error getting the folderName %s from the folderID %d: %s", folderName, folderID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	subfolderName, err := database.GetSubfolderNameFromID(s.Database, subfolderID)
	if err != nil {
		log.Error("Error getting the subfolderName %s from the subfolderID %d: %s", subfolderName, subfolderID, err)
		c.Status(http.StatusBadRequest)
		return
	}
	
	subfolderDetails, err := database.GetAllSubfolderDetailsForID(s.Database, subfolderID, folderID)
	if err != nil {
		errorMessage := fmt.Sprintf("Error retrieving the details for the subfolder %s: %s", subfolderName, err)
		log.Error(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": errorMessage,
		})
		return
	}	
	
	doesFolderExist, err := database.FolderExists(s.Database, folderName)
	if err != nil {
		log.Error("Error while checking if the folder already exists in db: %s", doesFolderExist)
		c.Status(http.StatusInternalServerError)
		return
	}
	if !doesFolderExist {
		log.Error("The folder with ID %d does not exist", folderID)
		c.Status(http.StatusBadRequest)
		return
	}

	doesSubfolderExist, err := database.SubfolderExists(s.Database, folderID, subfolderName)
	if err != nil {
		log.Error("Error while checking if the subfolder %s already exists in db: %s", subfolderName, doesSubfolderExist)
		c.Status(http.StatusInternalServerError)
		return
	}
	if !doesSubfolderExist {
		log.Error("The subfolder with ID %d does not exist", subfolderID)
		c.Status(http.StatusBadRequest)
		return
	}

	filesDetails, gsErr := database.GetAllFilesDetails(s.Database, folderID, subfolderID)
	if gsErr != nil {
		errorMessage := fmt.Sprintf("Error retrieving all files for the subfolder %s: %s", subfolderName, gsErr)
		log.Error(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": errorMessage,
		})
		return
	}

	log.Info("Successfully retrieved all files for the subfolder %s", subfolderName)
	c.JSON(http.StatusOK, gin.H{
		"files": filesDetails,
		"workspace" : folderName,
		"currentFolder": subfolderDetails.Name,
		"ownerID" : subfolderDetails.OwnerID,
		"isLocked" : subfolderDetails.IsLocked,
	})
}

func (s *Service) HandleGetFileForFileID(c *gin.Context) {
	fileID, err := getIntParameterFromRequest(c, "file_id")
	if err != nil {
		log.Error("Error retrieving file id parameter from the "+
			"HandleGetFileForFileID request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	subfolderID, err := getIntParameterFromRequest(c, "subfolder_id")
	if err != nil {
		log.Error("Error retrieving subfolder_id parameter from the "+
			"getIntParameterFromRequest request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	folderID, err := getIntParameterFromRequest(c, "folder_id")
	if err != nil {
		log.Error("Error retrieving folder_id parameter from the "+
			"HandleGetFileForFileID request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	fileDetails, gsErr := database.GetFilesDetailsForFileID(s.Database, fileID, folderID, subfolderID)
	if gsErr != nil {
		errorMessage := fmt.Sprintf("Error retrieving the file details for id %d: %s", fileID, gsErr)
		log.Error(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": errorMessage,
		})
		return
	}

	log.Info("Successfully retrieved details for the file with ID %d", fileID)
	c.JSON(http.StatusOK, gin.H{
		"file": fileDetails,
	})
}

func (s *Service) HandlePostCheckFilePassword(c *gin.Context) {
	var verifyPassword VerifyFilePassword
	err := c.BindJSON(&verifyPassword)
	if err != nil {
		log.Error("Error %s binding the JSON for HandlePostCheckFilePassword request %s", err, c.Request.Body)
		c.Status(http.StatusBadRequest)
		return
	}

	folderID, err := getIntParameterFromRequest(c, "folder_id")
	if err != nil {
		log.Error("Error retrieving folder_id parameter from the "+
			"HandlePostCheckFilePassword request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	subfolderID, err := getIntParameterFromRequest(c, "subfolder_id")
	if err != nil {
		log.Error("Error retrieving subfolder_id parameter from the "+
			"HandlePostCheckFilePassword request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	fileID, err := getIntParameterFromRequest(c, "file_id")
	if err != nil {
		log.Error("Error retrieving file id parameter from the request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	fileName, err := database.GetFilenameFromID(s.Database, fileID)
	if err != nil {
		log.Error("Error getting the file name from the fileID %d: %s", fileID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	isPasswordCorrect, err := database.VerifyFilePassword(s.Database, fileID, folderID, subfolderID, verifyPassword.Password)
	if err != nil {
		log.Error("Error while trying to verify the file password")
		c.Status(http.StatusInternalServerError)
		return
	}
	if !isPasswordCorrect {
		log.Error("The provided password for the file %s is not correct", fileName)
		c.Status(http.StatusUnauthorized)
		return
	}

	log.Info("The password provided by the user for subfolder %s is correct", fileName)
	c.Status(http.StatusOK)
}

func (s *Service) HandlePostModifiedFile(c *gin.Context) {
	fileID, err := getIntParameterFromRequest(c, "file_id")
	if err != nil {
		log.Error("Error retrieving file id parameter from the "+
			"HandlePostModifiedFile request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	subfolderID, err := getIntParameterFromRequest(c, "subfolder_id")
	if err != nil {
		log.Error("Error retrieving subfolder_id parameter from the "+
			"getIntParameterFromRequest request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	folderID, err := getIntParameterFromRequest(c, "folder_id")
	if err != nil {
		log.Error("Error retrieving folder_id parameter from the "+
			"HandlePostModifiedFile request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	folderName, err := database.GetFolderNameFromID(s.Database, folderID)
	if err != nil {
		log.Error("Error getting the folderName from the folderID %d: %s", folderID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	subfolderName, err := database.GetSubfolderNameFromID(s.Database, subfolderID)
	if err != nil {
		log.Error("Error getting the subfolderName %s from the subfolderID %d: %s", subfolderName, subfolderID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	filename, err := database.GetFilenameFromID(s.Database, fileID)
	if err != nil {
		log.Error("Error getting the filename for the fileID %d: %s", fileID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	buffer, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		errorMessage := fmt.Sprintln("Error while reading the file")
		log.Error(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": errorMessage,
		})
		return
	}

	filePath, err := filepath.Abs(pathToSaveFiles + folderName + "/" + subfolderName + "/" + filename )
	if err != nil {
		errorMessage := fmt.Sprintf("Error setting the file path %s", err)
		log.Error(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": errorMessage,
		})
		return
	 }
	file, err := os.Create(filePath)
	if err != nil {
		errorMessage := fmt.Sprintf("Error creating the updated file on the path %s", filePath)
		log.Error(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": errorMessage,
		})
		return
	 }
	defer file.Close()
	bytesWritten, _ := file.Write(buffer)

	log.Info("File %s successfully changed! Wrote %d bytes.", filename, bytesWritten)
	c.JSON(http.StatusOK, gin.H{
		"id": fileID,
	})
}

func (s *Service) HandleRemoveFile(c *gin.Context) {
	claims, err := verifyClaims(c)
	if err != nil {
		// if the claims not exist, mark it as unauthorised, otherwise, when the account is not activated,
		// just return, so the status code is 403, from the verifyClaims logic
		if err.Error() == ClaimsNotExist {
			log.Error("Error retrieving the claims from JWT")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": ClaimsNotExist,
			})
		}
		return
	}

	fileID, err := getIntParameterFromRequest(c, "file_id")
	if err != nil {
		log.Error("Error retrieving file id parameter from the request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	subfolderID, err := getIntParameterFromRequest(c, "subfolder_id")
	if err != nil {
		log.Error("Error retrieving subfolder_id parameter from the "+
			"getIntParameterFromRequest request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	folderID, err := getIntParameterFromRequest(c, "folder_id")
	if err != nil {
		log.Error("Error retrieving folder_id parameter from the request: %s", err)
		c.Status(http.StatusBadRequest)
		return
	}

	folderName, err := database.GetFolderNameFromID(s.Database, folderID)
	if err != nil {
		log.Error("Error getting the folder name from the folderID %d: %s", folderID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	subfolderName, err := database.GetSubfolderNameFromID(s.Database, subfolderID)
	if err != nil {
		log.Error("Error getting the subfolderName %s from the subfolderID %d: %s", subfolderName, subfolderID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	filename, err := database.GetFilenameFromID(s.Database, fileID)
	if err != nil {
		log.Error("Error getting the filename for the fileID %d: %s", fileID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	isFileDeleted := database.RemoveFile(s.Database, fileID, folderID, claims.Id, subfolderID)
	if !isFileDeleted {
		errorMessage := fmt.Sprintf("Error removing the file %s with ID %d", filename, fileID)
		log.Error(errorMessage)
		c.Status(http.StatusInternalServerError)
		return
	} else {
		err := os.Remove(pathToSaveFiles + "/" + folderName + "/" + subfolderName + "/" + filename)
		if err != nil {
			log.Fatal("%s", err)
		}
		log.Info("Successfully deleted the file %s from workspace %s subfolder %s", filename, folderName, subfolderName)
	}

	log.Info("Successfully removed file %s from the database", filename)
	c.Status(http.StatusOK)
}
