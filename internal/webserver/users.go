package webserver

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/CosminMocanu97/dissertationBackend/internal/auth"
	"github.com/CosminMocanu97/dissertationBackend/internal/database"
	"github.com/CosminMocanu97/dissertationBackend/internal/types"
	"github.com/CosminMocanu97/dissertationBackend/internal/utils"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
	//"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

const (
	ERROR_PASSWORD_IS_TOO_SHORT   = "the password is too short"
	ERROR_EMAIL_IS_INVALID        = "the email is not valid"
	ERROR_USER_ALREADY_EXISTS     = "the email already exists in the database"
	ERROR_INVALID_CREDENTIALS 	  = "invalid credentials"
	ERROR_USER_NOT_ACTIVATED 	  = "account is not activated"
)

//login contorller interface
type LoginController interface {
	Login(ctx *gin.Context) string
}

func Login(db *sql.DB, ctx *gin.Context) (int64, bool, map[string]string, error) {
	var credential types.LoginCredentials
	err := ctx.ShouldBind(&credential)
	if err != nil {
		log.Error("Could not bind the LoginCredentials for login")
		return 0, false, map[string]string{}, err
	}

	isAccountActivated, gsErr := database.UserIsActivated(db, credential.Email)
	if gsErr != nil {
		return 0, false, map[string]string{}, gsErr
	}
	if !isAccountActivated {
		return 0, false, map[string]string{}, gsErr
	}

	isUserAuthenticated, gsErr := database.VerifyLoginCredentials(db, credential.Email, credential.Password)
	if gsErr != nil {
		return 0, false, map[string]string{}, gsErr
	}
	if !isUserAuthenticated {
		gsErr = errors.New(ERROR_INVALID_CREDENTIALS)
		return 0, false, map[string]string{}, gsErr
	}

	jwtService := auth.JWTAuthService()
	// retrieve the id of the user that logged in
	user, gsErr := database.GetUserDetailsForEmail(db, credential.Email)
	if gsErr != nil {
		log.Error("Error retrieving the user details for email %s: %s", credential.Email, err)
		return 0, false, map[string]string{}, err
	}
	log.Info("Jwt token was successfully generated!")
	return user.ID, user.IsAdmin, jwtService.GenerateToken(user.ID, credential.Email, user.IsActivated), nil
}

// HandlePostRegisterRequest godoc
// @Summary register new user
// @Description register new user
// @Accept  json
// @Produce  json
// @Param  registrationData request body types.RegistrationData true "registration data"
// @Success 200 {json} http.StatusOK
// @Failure 400 {json} http.StatusBadRequest
// @Failure 500 {json} http.StatusInternalServerError
// @Failure default {json} http.StatusInternalServerError
// @Router /register [post]
func (s *Service) HandlePostRegisterRequest(c *gin.Context) {

	var registrationData types.RegistrationData
	err := c.ShouldBind(&registrationData)
	if err != nil {
		log.Error("Error binding the registration data: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Datele introduse nu sunt valide",
		})
		return
	}

	emailIsValid := utils.ValidateEmail(registrationData.Email)
	if !emailIsValid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ERROR_EMAIL_IS_INVALID,
		})
		return
	}

	passwordIsValid := utils.ValidatePassword(registrationData.Password)
	if !passwordIsValid {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": ERROR_PASSWORD_IS_TOO_SHORT,
		})
		return
	}

	activationToken := utils.GenerateRawAccountActivationToken()
	log.Info("The registrationData is email: %s, password: %s", registrationData.Email, registrationData.Password)

	userAdded, gsErr := database.AddNewUser(s.Database, registrationData, activationToken)
	if gsErr != nil {
		log.Error("Error adding the user %s with the password %s: %s", registrationData.Email, registrationData.Password, gsErr)
		c.JSON(http.StatusConflict, gin.H{
			"error": ERROR_USER_ALREADY_EXISTS,
		})
		return
	} else {
		// if the user wasn't successfully added
		if !userAdded {
			log.Error("The user %s with the password %s was not registered in the database: %s", registrationData.Email, registrationData.Password, gsErr)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "There is a problem on the server while trying to add the user",
			})
			return
		} else {
			// get the id of the latest user added to the database
			user, gsErr := database.GetUserDetailsForEmail(s.Database, registrationData.Email)
			if gsErr != nil {
				gsErr = database.RemoveUser(s.Database, registrationData.Email)
				log.Error("Error retrieving the ID for email %s: %s", registrationData.Email, gsErr)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "There is a problem on the server while retriving the id",
				})
				return
			}

			activationToken := utils.BuildActivationTokenWithUserId(user.ID, activationToken)
			// send activation email
			recipients := []string{registrationData.Email}
			err = s.MailingService.SendEmail(recipients, "Activare cont", activationToken, "Accesati linkul pentru a valida contul " + "http://localhost:3000/activate/"+ activationToken)
			if err != nil {
				// if the activation email fails, remove the user
				gsErr = database.RemoveUser(s.Database, registrationData.Email)
				if gsErr != nil {
					log.Error("Error removing the user from the database if the activation email sending fails: %s", gsErr)
				}
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "We couldn't send the email at the specified address",
				})
				return
			}
			c.JSON(http.StatusOK, nil)
		}
	}
}

// Login godoc
// @Summary login
// @Description login user by credentials
// @Accept  json
// @Produce  json
// @Param  LoginCredentials request body types.LoginCredentials true "login data"
// @Success 200 {object} types.LoginResponse
// @Failure 400 {json} http.StatusBadRequest
// @Failure 401 {json} http.StatusUnauthorized
// @Failure 500 {json} http.StatusInternalServerError
// @Failure default {json} http.StatusInternalServerError
// @Router /login [post]
func (s *Service) HandlePostLoginRequest(c *gin.Context) {
	// actual logic
	id, isAdmin, tokens, err := Login(s.Database, c)
	if err != nil {
		// if we have an error check if it is a functional or a logical one
		if err == sql.ErrNoRows {
			c.Status(http.StatusExpectationFailed)
			return
		} else if err.Error() == ERROR_INVALID_CREDENTIALS {
			c.Status(http.StatusBadRequest)
			return
		} else if err.Error() == ERROR_USER_NOT_ACTIVATED {
			c.Status(http.StatusForbidden)
			return
		} else {
			log.Error("%s", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Theres a problem on the server while logging in",
			})
		}
	} else {
		c.JSON(http.StatusOK, gin.H {
			"id":    id,
			"admin" : isAdmin,
			"token": tokens["access_token"],
			"refresh_token" : tokens["refresh_token"],
		})
	}
}

// handle Activate Account
// HandlePostActivateAccount godoc
// @Summary activate account
// @Description activate account by token
// @Accept  json
// @Produce  json
// @Param token query string true "user token"
// @Success 200 {json} http.StatusOK
// @Failure 400 {json} http.StatusBadRequest
// @Failure 401 {json} http.StatusUnauthorized
// @Failure 500 {json} http.StatusInternalServerError
// @Failure default {json} http.StatusInternalServerError
// @Router /activate/{token} [get]
func (s *Service) HandlePostActivateAccount(c *gin.Context) {
	rawActivationToken := c.Params.ByName("token")
	// if the parameter doesn't exist, return StatusBadRequest
	if len(rawActivationToken) == 0 {
		log.Error("Error retrieving the activation token parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Your request is not valid",
		})
		return
	}

	userID, activationToken, err := utils.GetUserIDAndActivationTokenFromRawActivationToken(rawActivationToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "The verification failed, please try again",
		})
		return
	} else {
		tokenIsCorrect, gsErr := database.VerifyActivationToken(s.Database, userID, activationToken)
		if gsErr != nil {
			log.Error("Error validating the activation token: %s", gsErr)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid activation token",
			})
			return
		} else {
			if tokenIsCorrect {
				gsErr = database.ActivateAccount(s.Database, userID)
				if gsErr != nil {
					log.Error("Error activating the account for ID %d: %s", userID, gsErr)
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": "Error while trying to activate the account",
					})
					return
				}
				// after the account is activated, renew de activation token
				newActivationToken := utils.GenerateRawAccountActivationToken()
				gsErr := database.RenewActivationToken(s.Database, userID, newActivationToken)
				if gsErr != nil {
					log.Error("Error renewing the activation token for userID %d after activating the account: %s", userID, gsErr)
				}
				log.Info("Successfully activated account with ID %d", userID)
				c.JSON(http.StatusOK, gin.H{
					"message": "The account was successfully activated",
				})
				return
			} else {
				log.Error("Error activating the account with ID %d: invalid token %s", userID, activationToken)
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": err,
				})
			}
		}
	}
}


//HandlePostForgotPasswordRequest handles POST "/forgot-password" request
func (s *Service) HandlePostForgotPasswordRequest(c *gin.Context) {

	var registrationData types.RegistrationData
	err := c.ShouldBind(&registrationData)
	if err != nil {
		log.Error("Error binding the registration data: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	newToken := utils.GenerateRawAccountActivationToken()
	user, gsErr := database.GetUserDetailsForEmail(s.Database, registrationData.Email)
	if gsErr != nil {
		log.Error("Error retrieving the user details for email %s: %s", registrationData.Email, gsErr)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gsErr,
		})
		return
	} else {
		newTokenWithUserId := utils.BuildActivationTokenWithUserId(user.ID, newToken)
		gsErr = database.RenewActivationToken(s.Database, user.ID, newToken)
		if gsErr != nil {
			log.Error("UUID: %s; Error changing the token for the account with ID %d: %s", user.ID, gsErr)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gsErr,
			})
			return
		}

		// send password renewal email
		recipients := []string{registrationData.Email}
		err = s.MailingService.SendEmail(recipients, "Resetati parola", newTokenWithUserId, "http://localhost:3000/renew-password/" + newTokenWithUserId)
		if err != nil {
			log.Error("Error sending password renewal email for the user with ID %d: %s", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Error sending password renewal email",
			})
			return
		} else {
			c.Status(http.StatusOK)
		}
	}
}

//HandlePostRenewPasswordRequest handles POST "/renew-password" request
func (s *Service) HandlePostRenewPasswordRequest(c *gin.Context) {

	rawActivationToken := c.Params.ByName("token")
	// if the parameter does not exist, return StatusBadRequest
	if len(rawActivationToken) == 0 {
		log.Error("Error retrieving the activation token parameter from the HandlePostActivateAccount request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Error retrieving the activation token parameter from the request",
		})
		return
	}

	var registrationData types.RegistrationData
	err := c.ShouldBind(&registrationData)
	if err != nil {
		log.Error("Error binding the registration data: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	userID, activationToken, err := utils.GetUserIDAndActivationTokenFromRawActivationToken(rawActivationToken)
	if err != nil {
		log.Error("Error retrieving userID from raw activation token %s: %s", rawActivationToken, err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token format",
		})
	} else {
		tokenIsCorrect, gsErr := database.VerifyActivationToken(s.Database, userID, activationToken)
		if gsErr != nil {
			log.Error("UUID: %s; Error validating the token for password update: %s", gsErr)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token for password update",
			})
		} else {
			if tokenIsCorrect {
				gsErr = database.UpdatePassword(s.Database, userID, activationToken, registrationData.Password)
				if gsErr != nil {
					log.Error("Error updating the password for the account with ID %d: %s", userID, gsErr)
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": gsErr,
					})
				} else {
					// renew the account token co it can't be reused again
					newToken := utils.GenerateRawAccountActivationToken()
					gsErr = database.RenewActivationToken(s.Database, userID, newToken)
					if gsErr != nil {
						log.Error("Error renewing the token for userID %d after renewing their password: %s", userID, gsErr)
					}
					log.Info("The password was successfully changed!")
					c.JSON(http.StatusOK, gin.H{
						"message": "The password was successfully updated",
					})
				}
			} else {
				log.Error("Error changing the password for the account with ID %d: invalid token %s", userID, activationToken)
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": gsErr,
				})
			}
		}
	}
}
