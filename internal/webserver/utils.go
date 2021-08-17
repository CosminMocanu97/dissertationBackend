package webserver

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/CosminMocanu97/dissertationBackend/internal/auth"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
)


// getClaimsFromMiddleware returns true if the key "claims" exists, alongside the claims, and false otherwise
func getClaimsFromMiddleware(c *gin.Context) (bool, *auth.AuthCustomClaims) {
	val, keyExists := c.Get("claims")
	claims := val.(*auth.AuthCustomClaims)
	return keyExists, claims
}

func verifyClaims(c *gin.Context) (*auth.AuthCustomClaims, error) {
	claimsExists, claims := getClaimsFromMiddleware(c)
	if !claimsExists {
		log.Error("Error retrieving the claims from JWT")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Error retrieving the claims from JWT",
		})
		return nil, errors.New("error retrieving the claims from the JWT")
	}

	// if the account is not activated, the user doesn't have the right to perform any operations
	if !claims.IsActivated {
		c.AbortWithStatus(http.StatusExpectationFailed)
		return nil, errors.New("account is not activated")
	}

	return claims, nil
}

func getIntParameterFromRequest(c *gin.Context, paramName string) (int64, error) {
	// if the parameter doesn't exist, it will be an empty string
	rawParam := c.Params.ByName(paramName)
	if len(rawParam) == 0 {
		errorMessage := fmt.Sprintf("Error retrieving the parameter %s from the request", paramName)
		log.Error("UUID: %s; %s", errorMessage)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": errorMessage,
		})
		return 0, errors.New(errorMessage)
	}

	intParam, err := strconv.Atoi(rawParam)
	if err != nil {
		log.Error("Error converting the parameter %s to an int: %s", rawParam, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User id couldnt be fetched",
		})
	}
	param := int64(intParam)
	return param, nil
}
