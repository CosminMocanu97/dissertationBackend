package webserver

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/CosminMocanu97/dissertationBackend/internal/database"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
)

var ( 
	ClaimsNotExist = "Error retrieving the claims from the JWT"
	NAME_IS_EMPTY = "folder name cannot be empty"
	FOLDER_ALREADY_EXISTS = "the folder already exists in the database"
)

type Folder struct {
	Name string `json:"folderName"`
	Password string `json:"password"`
}

type VerifyPassword struct {
	Password string `json:"password"`
}

func (s *Service) HandlePostFolderRequest(c *gin.Context) {
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

	var folderDetails Folder
	var isLocked bool
	err = c.BindJSON(&folderDetails)
	if err != nil {
		log.Error("Error %s binding the JSON for the add new folder request %s", err, c.Request.Body)
		c.Status(http.StatusBadRequest)
		return
	}
	if len(folderDetails.Password) == 0 {
		isLocked = false
	} else {
		isLocked = true
	}

	folderErr := os.Mkdir(pathToSaveFiles + folderDetails.Name, 0755)
	if folderErr != nil {
		errorMessage := fmt.Sprintf("Error creating the folder on file system for path %s", pathToSaveFiles)
		log.Error(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errorMessage,
		})
		return
	}

	//add folder to database and get the id
	folderId, gsErr := database.AddNewFolder(s.Database, claims.Id, folderDetails.Name, folderDetails.Password, isLocked)
	if gsErr != nil {
		errorMessage := fmt.Sprintf("Error creating the folder %s: %s", folderDetails.Name, gsErr)
		log.Error(errorMessage)
		if gsErr.Error() == FOLDER_ALREADY_EXISTS {
			c.JSON(http.StatusConflict, gin.H{
				"error": errorMessage,
			})
			return
		} else if gsErr.Error() == NAME_IS_EMPTY {
			c.JSON(http.StatusForbidden, gin.H{
				"error": errorMessage,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": errorMessage,
		})
		return
	} 

	log.Info("Successfully created the folder with the name %s", folderDetails.Name)
	c.JSON(http.StatusOK, gin.H{
		"id": folderId,
	})
}

func (s *Service) HandleGetAllFullFolderDetails(c *gin.Context) {
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

	folderDetails, gsErr := database.GetAllFoldersDetails(s.Database)
	if gsErr != nil {
		errorMessage := fmt.Sprintf("Error retrieving all folders: %s", gsErr)
		log.Error(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": errorMessage,
		})
		return
	}

	log.Info("Successfully retrieved the list of all folders for userID %d", claims.Id)
	c.JSON(http.StatusOK, gin.H{
		"folders": folderDetails,
	})
}

func (s *Service) HandlePostCheckPassword(c *gin.Context) {
	var verifyPassword VerifyPassword
	err := c.BindJSON(&verifyPassword)
	if err != nil {
		log.Error("Error %s binding the JSON for the verify folder password request %s", err, c.Request.Body)
		c.Status(http.StatusBadRequest)
		return
	}

	folderID, err := getIntParameterFromRequest(c, "folder_id")
	if err != nil {
		log.Error("Error retrieving folder_id parameter from the "+
			"HandlePostCheckPassword request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	isPasswordCorrect, err := database.VerifyFolderPassword(s.Database, folderID, verifyPassword.Password)
	if err != nil {
		log.Error("Error while trying to verify the folder password")
		c.Status(http.StatusInternalServerError)
		return
	}
	if !isPasswordCorrect {
		log.Error("The provided password for the folder with ID %d is not correct", folderID)
		c.Status(http.StatusUnauthorized)
		return
	}

	log.Info("The password provided by the user is correct")
	c.Status(http.StatusOK)
}

func (s *Service) HandleRemoveFolder(c *gin.Context) {
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
		log.Error("Error retrieving folder_id parameter from the " +
			"getIntParameterFromRequest request: %s", err)
		c.Status(http.StatusBadRequest)
		return
	}

	folderName, err := database.GetFolderNameFromID(s.Database, folderID)
	if err != nil {
		log.Error("Error getting the folderName %s from the folderID %d: %s", folderName, folderID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	folderErr := database.RemoveFolder(s.Database, folderID, claims.Id)
	filesErr := database.RemoveFilesFromFolder(s.Database, folderID, claims.Id)

	if !folderErr {
		errorMessage := fmt.Sprintf("Error removing the folder with ID %d", folderID)
		log.Error(errorMessage)
		c.Status(http.StatusInternalServerError)
		return
	} else if !filesErr {
		errorMessage := fmt.Sprintf("Error removing the files from folder %s", folderName)
		log.Error(errorMessage)
		c.Status(http.StatusInternalServerError)
		return
	} else {
		err := os.RemoveAll(pathToSaveFiles + "/" + folderName)
		if err != nil {
			log.Fatal("%s", err)
		}
		log.Info("Successfully deleted the folder %s from file system", folderName)
	}

	log.Info("Successfully removed folder %s from the database", folderName)
	c.Status(http.StatusOK)
}
