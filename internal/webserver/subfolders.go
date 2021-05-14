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
	pathToSaveSubfolders = "/home/cosminel/DissertationAppFolders/"
	SUBFOLDER_NAME_IS_EMPTY = "subfolder name cannot be empty"
	SUBFOLDER_ALREADY_EXISTS = "this subfolder already exists in the database"
)

type Subfolder struct {
	Name string `json:"subfolderName"`
	Password string `json:"password"`
}

type VerifyPasswordSubfolder struct {
	Password string `json:"password"`
}

func (s *Service) HandlePostSubfolderRequest(c *gin.Context) {
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

	folderName, err := database.GetFolderNameFromID(s.Database, folderID)
	if err != nil {
		log.Error("Error getting the folderName %s from the folderID %d: %s", folderName, folderID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	subfolderPath := pathToSaveSubfolders + folderName + "/"
	var subfolderDetails Subfolder
	var isLocked bool
	err = c.BindJSON(&subfolderDetails)
	if err != nil {
		log.Error("Error %s binding the JSON for the add new subfolder request %s", err, c.Request.Body)
		c.Status(http.StatusBadRequest)
		return
	}
	if len(subfolderDetails.Password) == 0 {
		isLocked = false
	} else {
		isLocked = true
	}

	subfolderId, gsErr := database.AddNewSubfolder(s.Database, claims.Id, folderID, subfolderDetails.Name, subfolderDetails.Password, isLocked)
	if gsErr != nil {
		errorMessage := fmt.Sprintf("Error creating the subfolder %s: %s", subfolderDetails.Name, gsErr)
		log.Error(errorMessage)
		if gsErr.Error() == SUBFOLDER_ALREADY_EXISTS {
			c.JSON(http.StatusConflict, gin.H{
				"error": errorMessage,
			})
			return
		} else if gsErr.Error() == SUBFOLDER_NAME_IS_EMPTY {
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

	subfolderErr := os.Mkdir(subfolderPath + subfolderDetails.Name, 0755)
	if subfolderErr != nil {
		errorMessage := fmt.Sprintf("Error creating the subfolder on file system for path %s : subfolder already exists", subfolderPath)
		log.Error(errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errorMessage,
		})
		return
	}

	log.Info("Successfully created the subfolder with the name %s in folder %s", subfolderDetails.Name, folderName)
	c.JSON(http.StatusOK, gin.H{
		"id": subfolderId,
	})
}

func (s *Service) HandleGetAllFullSubfolderDetails(c *gin.Context) {
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

	folderName, err := database.GetFolderNameFromID(s.Database, folderID)
	if err != nil {
		log.Error("Error getting the folderName %s from the folderID %d: %s", folderName, folderID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	folderDetails, err := database.GetAllFoldersDetailsForID(s.Database, folderID)
	if err != nil {
		errorMessage := fmt.Sprintf("Error retrieving the details for the folder %d: %s", folderID, err)
		log.Error(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": errorMessage,
		})
		return
	}	

	subfolderDetails, gsErr := database.GetAllSubFoldersDetails(s.Database, folderID)
	if gsErr != nil {
		errorMessage := fmt.Sprintf("Error retrieving all subfolders from %s: %s", folderName, gsErr)
		log.Error(errorMessage)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": errorMessage,
		})
		return
	}

	log.Info("Successfully retrieved the list of all subfolders from %s for userID %d", folderName, claims.Id)
	c.JSON(http.StatusOK, gin.H{
		"subfolders": subfolderDetails,
		"rootfolder" : folderDetails.Name,
		"ownerID" : folderDetails.OwnerID,
	})
}

func (s *Service) HandlePostCheckPasswordSubfolder(c *gin.Context) {
	var verifyPassword VerifyPasswordSubfolder
	err := c.BindJSON(&verifyPassword)
	if err != nil {
		log.Error("Error %s binding the JSON for HandlePostCheckPasswordSubfolder request %s", err, c.Request.Body)
		c.Status(http.StatusBadRequest)
		return
	}


	subfolderID, err := getIntParameterFromRequest(c, "subfolder_id")
	if err != nil {
		log.Error("Error retrieving subfolder_id parameter from the "+
			"HandlePostCheckPasswordSubfolder request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	folderID, err := getIntParameterFromRequest(c, "folder_id")
	if err != nil {
		log.Error("Error retrieving folder_id parameter from the "+
			"HandlePostCheckPasswordSubfolder request: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	subfolderName, err := database.GetSubfolderNameFromID(s.Database, subfolderID)
	if err != nil {
		log.Error("Error getting the subfolderName %s from the subfolderID %d: %s", subfolderName, subfolderID, err)
		c.Status(http.StatusBadRequest)
		return
	}

	isPasswordCorrect, err := database.VerifySubfolderPassword(s.Database, subfolderID, folderID, verifyPassword.Password)
	if err != nil {
		log.Error("Error while trying to verify the subfolder password")
		c.Status(http.StatusInternalServerError)
		return
	}
	if !isPasswordCorrect {
		log.Error("The provided password for the subfolder %s is not correct", subfolderName)
		c.Status(http.StatusUnauthorized)
		return
	}

	log.Info("The password provided by the user for subfolder %s is correct", subfolderName)
	c.Status(http.StatusOK)
}

func (s *Service) HandleRemoveSubfolder(c *gin.Context) {
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

	subfolderID, err := getIntParameterFromRequest(c, "subfolder_id")
	if err != nil {
		log.Error("Error retrieving subfolder_id parameter from the "+
			"HandlePostCheckPasswordSubfolder request: %s", err)
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

	subfolderErr := database.RemoveSubfolder(s.Database, subfolderID, folderID, claims.Id)
	filesErr := database.RemoveFilesFromSubfolder(s.Database, subfolderID, folderID, claims.Id)

	if !subfolderErr {
		errorMessage := fmt.Sprintf("Error removing the subfolder %s from folder %s", subfolderName, folderName)
		log.Error(errorMessage)
		c.Status(http.StatusInternalServerError)
		return
	} else if !filesErr {
		errorMessage := fmt.Sprintf("Error removing the files from subfolder %s", subfolderName)
		log.Error(errorMessage)
		c.Status(http.StatusInternalServerError)
		return
	} else {
		err := os.RemoveAll(pathToSaveFiles + folderName + "/" + subfolderName)
		if err != nil {
			log.Fatal("%s", err)
		}
		log.Info("Successfully deleted the subfolder %s from file system", subfolderName)
	}

	log.Info("Successfully removed subfolder %s from the database", subfolderName)
	c.Status(http.StatusOK)
}
