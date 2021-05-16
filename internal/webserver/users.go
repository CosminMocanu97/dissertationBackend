package webserver

import (
	"database/sql"
	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	"github.com/CosminMocanu97/dissertationBackend/internal/auth"
	"github.com/CosminMocanu97/dissertationBackend/internal/database"
	"github.com/CosminMocanu97/dissertationBackend/internal/types"
	"github.com/CosminMocanu97/dissertationBackend/internal/utils"
	"github.com/CosminMocanu97/dissertationBackend/pkg/gserror"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
	"net/http"
)

const (
	ERROR_PASSWORD_IS_TOO_SHORT   = "Parola introdusa este prea scurta"
	ERROR_EMAIL_IS_INVALID        = "Emailul este invalid"
)

func Login(db *sql.DB, ctx *gin.Context, id snowflake.ID) (int64, string, gserror.GSError) {
	var credential types.LoginCredentials
	err := ctx.ShouldBind(&credential)
	if err != nil {
		log.Error("UUID: %s; Could not bind the LoginCredentials for login", id)
		return 0, "", gserror.NewInternalGSError(err)
	}

	isAccountActivated, gsErr := database.UserIsActivated(db, credential.Email, id)
	if gsErr != gserror.NoError {
		return 0, "", gsErr
	}
	if !isAccountActivated {
		return 0, "", gserror.NewGSError(nil, database.ERROR_USER_NOT_ACTIVATED, database.ERROR_USER_NOT_ACTIVATED)
	}

	isUserAuthenticated, gsErr := database.VerifyLoginCredentials(db, credential.Email, credential.Password, id)
	if gsErr != gserror.NoError {
		return 0, "", gsErr
	}
	if !isUserAuthenticated {
		return 0, "", gserror.NewGSError(nil, database.ERROR_INVALID_CREDENTIALS, database.ERROR_INVALID_CREDENTIALS_TRANSLATION)
	}

	jwtService := auth.JWTAuthService()
	// retrieve the id of the user that logged in
	user, gsErr := database.GetUserDetailsForEmail(db, credential.Email, id)
	if gsErr != gserror.NoError {
		log.Error("UUID: %s; Error retrieving the user details for email %s: %s", id, credential.Email, err)
		return 0, "", gserror.NewInternalGSError(err)
	}
	return user.ID, jwtService.GenerateToken(user.ID, credential.Email, user.IsActivated), gserror.NoError
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
	snowflakeID := node.Generate()

	var registrationData types.RegistrationData
	err := c.ShouldBind(&registrationData)
	if err != nil {
		log.Error("UUID: %s; Error binding the registration data: %s", snowflakeID, err)
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
	log.Info("UUID: %s; The registrationData is user: %s, password: %s, phone: %s",
		snowflakeID, registrationData.Email, registrationData.Password)
	userAdded, gsErr := database.AddNewUser(s.Database, registrationData, activationToken, snowflakeID)
	if gsErr != gserror.NoError {
		log.Error("UUID: %s; Error adding the user %s with the password %s and phone number %s: %s",
			snowflakeID, registrationData.Email, registrationData.Password, gsErr)
		c.JSON(http.StatusConflict, gin.H{
			"error": gsErr.TranslatedHumanErrorMessage,
		})
		return
	} else {
		// if the user wasn't successfully added
		if !userAdded {
			log.Error("UUID: %s; The user %s with the password %s and phone number %s was not registered in the database: %s",
				snowflakeID, registrationData.Email, registrationData.Password, gsErr)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gsErr.TranslatedHumanErrorMessage,
			})
			return
		} else {
			// get the id of the latest user added to the database
			user, gsErr := database.GetUserDetailsForEmail(s.Database, registrationData.Email, snowflakeID)
			if gsErr != gserror.NoError {
				log.Error("UUID: %s; Error retrieving the ID for email %s: %s", snowflakeID, registrationData.Email, gsErr)
				gsErr = database.RemoveUser(s.Database, registrationData.Email, snowflakeID)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gsErr.TranslatedHumanErrorMessage,
				})
				return
			}

			activationToken := utils.BuildActivationTokenWithUserId(user.ID, activationToken)
			// send activation email
			recipients := []string{registrationData.Email}
			err = s.MailingService.SendEmail(recipients, "Activare cont", activationToken, "Accesati linkul pentru a valida contul " + "http://localhost:3000/activate/"+ activationToken, snowflakeID)
			if err != nil {
				// if the activation email fails, remove the user
				gsErr = database.RemoveUser(s.Database, registrationData.Email, snowflakeID)
				if gsErr != gserror.NoError {
					log.Error("UUID %s; Error removing the user from the database if the activation email sending fails: %s",
						snowflakeID, gsErr)
				}
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Nu am putut trimite email catre adresa dumneavoastra; va rugam verificati daca adresa este corecta",
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
	snowflakeID := node.Generate()
	// actual logic
	id, token, gsErr := Login(s.Database, c, snowflakeID)
	if gsErr != gserror.NoError {
		// if we have an error check if it is a functional or a logical one
		if gsErr.Err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gsErr.TranslatedHumanErrorMessage,
			})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gsErr.TranslatedHumanErrorMessage,
			})
		}
	} else {
		c.JSON(http.StatusOK, gin.H{
			"id":    id,
			"token": token,
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
	snowflakeID := node.Generate()
	rawActivationToken := c.Params.ByName("token")
	// if the parameter doesn't exist, return StatusBadRequest
	if len(rawActivationToken) == 0 {
		log.Error("UUID: %s; Error retrieving the activation token parameter", snowflakeID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cererea dumneavoastra de activare a contului nu este valida",
		})
		return
	}

	userID, activationToken, err := utils.GetUserIDAndActivationTokenFromRawActivationToken(rawActivationToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Verificarea nu a reusit, va rugam reincercati",
		})
		return
	} else {
		tokenIsCorrect, gsErr := database.VerifyActivationToken(s.Database, userID, activationToken, snowflakeID)
		if gsErr != gserror.NoError {
			log.Error("UUID: %s; Error validating the activation token: %s", snowflakeID, err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid activation token",
			})
			return
		} else {
			if tokenIsCorrect {
				gsErr = database.ActivateAccount(s.Database, userID, snowflakeID)
				if gsErr != gserror.NoError {
					log.Error("UUID: %s; Error activating the account for ID %d: %s", snowflakeID, userID, err)
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": gsErr.TranslatedHumanErrorMessage,
					})
					return
				}
				// after the account is activated, renew de activation token
				newActivationToken := utils.GenerateRawAccountActivationToken()
				gsErr := database.RenewActivationToken(s.Database, userID, newActivationToken, snowflakeID)
				if gsErr != gserror.NoError {
					log.Error("UUID: %s; Error renewing the activation token for userID %d after activating the account: %s",
						snowflakeID, userID, gsErr.TranslatedHumanErrorMessage)
				}
				log.Info("UUID: %s; Successfully activated account with ID %d", snowflakeID, userID)
				c.JSON(http.StatusOK, gin.H{
					"message": "The account was successfully activated",
				})
				return
			} else {
				log.Error("UUID: %s; Error activating the account with ID %d: invalid token %s",
					snowflakeID, userID, activationToken)
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": err,
				})
			}
		}
	}
}


//HandlePostForgotPasswordRequest handles POST "/forgot-password" request
func (s *Service) HandlePostForgotPasswordRequest(c *gin.Context) {
	// generate ID for the request
	snowflakeID := node.Generate()

	var registrationData types.RegistrationData
	err := c.ShouldBind(&registrationData)
	if err != nil {
		log.Error("UUID: %s; Error binding the registration data: %s", snowflakeID, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	newToken := utils.GenerateRawAccountActivationToken()
	user, gsErr := database.GetUserDetailsForEmail(s.Database, registrationData.Email, snowflakeID)
	if gsErr != gserror.NoError {
		log.Error("UUID: %s; Error retrieving the user details for email %s: %s",
			snowflakeID, registrationData.Email, gsErr)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gsErr.TranslatedHumanErrorMessage,
		})
		return
	} else {
		newTokenWithUserId := utils.BuildActivationTokenWithUserId(user.ID, newToken)
		gsErr = database.RenewActivationToken(s.Database, user.ID, newToken, snowflakeID)
		if gsErr != gserror.NoError {
			log.Error("UUID: %s; Error changing the token for the account with ID %d: %s",
				snowflakeID, user.ID, gsErr)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gsErr.TranslatedHumanErrorMessage,
			})
			return
		}

		// send password renewal email
		recipients := []string{registrationData.Email}
		err = s.MailingService.SendEmail(recipients, "Resetati parola", newTokenWithUserId, "http://localhost:3000/renew-password/" + newTokenWithUserId, snowflakeID)
		if err != nil {
			log.Error("UUID: %s; Error sending password renewal email for the user with ID %d: %s",
				snowflakeID, user.ID, err)
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
	// generate ID for the request
	snowflakeID := node.Generate()

	rawActivationToken := c.Params.ByName("token")
	// if the parameter does not exist, return StatusBadRequest
	if len(rawActivationToken) == 0 {
		log.Error("UUID: %s; Error retrieving the activation token parameter from the HandlePostActivateAccount request", snowflakeID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Error retrieving the activation token parameter from the request",
		})
		return
	}

	var registrationData types.RegistrationData
	err := c.ShouldBind(&registrationData)
	if err != nil {
		log.Error("UUID: %s; Error binding the registration data: %s", snowflakeID, err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
		return
	}

	userID, activationToken, err := utils.GetUserIDAndActivationTokenFromRawActivationToken(rawActivationToken)
	if err != nil {
		log.Error("UUID: %s; Error retrieving userID from raw activation token %s: %s",
			snowflakeID, rawActivationToken, err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token format",
		})
	} else {
		tokenIsCorrect, gsErr := database.VerifyActivationToken(s.Database, userID, activationToken, snowflakeID)
		if gsErr != gserror.NoError {
			log.Error("UUID: %s; Error validating the token for password update: %s", snowflakeID, err)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token for password update",
			})
		} else {
			if tokenIsCorrect {
				gsErr = database.UpdatePassword(s.Database, userID, activationToken, registrationData.Password, snowflakeID)
				if gsErr != gserror.NoError {
					log.Error("UUID: %s; Error updating the password for the account with ID %d: %s",
						snowflakeID, userID, gsErr)
					c.JSON(http.StatusInternalServerError, gin.H{
						"error": gsErr.TranslatedHumanErrorMessage,
					})
				} else {
					// renew the account token co it can't be reused again
					newToken := utils.GenerateRawAccountActivationToken()
					gsErr = database.RenewActivationToken(s.Database, userID, newToken, snowflakeID)
					if gsErr != gserror.NoError {
						log.Error("UUID: %s; Error renewing the token for userID %d after renewing their password: %s",
							snowflakeID, userID, gsErr)
					}
					c.JSON(http.StatusOK, gin.H{
						"message": "The password was successfully updated",
					})
				}
			} else {
				log.Error("UUID: %s; Error changing the password for the account with ID %d: invalid token %s",
					snowflakeID, userID, activationToken)
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": gsErr.TranslatedHumanErrorMessage,
				})
			}
		}
	}
}
