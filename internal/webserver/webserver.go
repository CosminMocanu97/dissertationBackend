package webserver

import (
	"fmt"
	"net/http"
	"os"
	"github.com/dgrijalva/jwt-go"

	"github.com/CosminMocanu97/dissertationBackend/internal/auth"
	"github.com/CosminMocanu97/dissertationBackend/internal/database"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
	"github.com/gin-gonic/gin"
)

type tokenReqBody struct {
	RefreshToken string `json:"refresh_token"`
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func Api(s *Service) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(CORS())

	// Routes
	// used for CORS
	r.GET("/", func(context *gin.Context) {
		context.Status(http.StatusOK)
	})
	r.Static("/getFile", "/home/cosminel/DissertationAppFolders/")
	//authentication
	r.GET("/ping", s.HandleGetPingRequest)
	r.POST("/register", s.HandlePostRegisterRequest)
	r.POST("/login", s.HandlePostLoginRequest)
	r.GET("/activate/:token", s.HandlePostActivateAccount)
	r.POST("/forgot-password", s.HandlePostForgotPasswordRequest)
	r.POST("/renew-password/:token", s.HandlePostRenewPasswordRequest)

	//folders endpoints
	r.GET("/user", AuthorizeJWT(), s.HandleGetAllFullFolderDetails)
	r.POST("/new_folder", AuthorizeJWT(), s.HandlePostFolderRequest)
	r.DELETE("/user/:folder_id/remove_folder", AuthorizeJWT(), s.HandleRemoveFolder)

	//subfolder endpoints
	r.GET("/user/:folder_id", AuthorizeJWT(), s.HandleGetAllFullSubfolderDetails)
	r.POST("/user/:folder_id/new_subfolder", AuthorizeJWT(), s.HandlePostSubfolderRequest)
	r.POST("/user/:folder_id/:subfolder_id", AuthorizeJWT(), s.HandlePostCheckPasswordSubfolder)
	r.DELETE("/user/:folder_id/:subfolder_id/remove_subfolder", AuthorizeJWT(), s.HandleRemoveSubfolder)

	//files endpoints
	r.GET("/user/:folder_id/:subfolder_id", AuthorizeJWT(), s.HandleGetAllFilesForCurrentFolder)
	r.GET("/user/:folder_id/:subfolder_id/:file_id", AuthorizeJWT(), s.HandleGetFileForFileID)
	r.POST("/user/:folder_id/:subfolder_id/:file_id", AuthorizeJWT(), s.HandlePostCheckFilePassword)
	r.POST("/user/:folder_id/:subfolder_id/upload", AuthorizeJWT(), s.HandlePostAddFile)
	r.POST("/user/:folder_id/:subfolder_id/:file_id/update", AuthorizeJWT(), s.HandlePostModifiedFile)
	r.DELETE("/user/:folder_id/:subfolder_id/:file_id/remove_file", AuthorizeJWT(), s.HandleRemoveFile)

	//generate new jwt
	r.POST("/newtoken", s.GenerateNewToken)

	return r
}

// JWT authorisation middleware
func AuthorizeJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		// if the token doesn't exist, return unauthorised
		if len(tokenString) == 0  {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "No authorization token in the request header",
			})
		}
		token, err := auth.JWTAuthService().ValidateToken(tokenString)
		switch err {
		case auth.NoJWTTokenProvidedError:
			log.Error("Unable to retrieve the Authorization Header: No JWT token was provided")
			c.AbortWithStatus(http.StatusForbidden)
			
		case auth.TokenIsExpired:
			log.Error("Unable to finish the request: %s", auth.TokenIsExpired)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "jwtExpired",
			})
			c.AbortWithStatus(http.StatusUnauthorized)
			
		case nil:
			if token.Valid {
				claims := token.Claims.(*auth.AuthCustomClaims)
				// if the account is not activated, abort
				if !claims.IsActivated {
					c.JSON(http.StatusUnauthorized, gin.H{
						"error": "The account is not activated",
					})
				} 
				c.Set("claims", claims)
				c.Next()
			} else {
				log.Error("the JWT token %s is invalid: %s", token.Raw, err)
				c.AbortWithStatus(http.StatusForbidden)
			}
		
		default:
			log.Error("Error validating the JWT token %s: %s", tokenString, err)
			c.AbortWithStatus(http.StatusForbidden)
		}
	}
}

func (s *Service) HandleGetPingRequest(c *gin.Context) {
	
	log.Info("Request to GET /ping")
	c.String(http.StatusOK, "pong")
}

func (s *Service) GenerateNewToken(c *gin.Context)  {
	var tokenReq tokenReqBody
	c.BindJSON(&tokenReq)

	var customClaims auth.RefreshAuthCustomClaims
	var secretKey = os.Getenv("JWT_SECRET")

	token, err := jwt.ParseWithClaims(tokenReq.RefreshToken, &customClaims, func(token *jwt.Token) (interface{}, error) {
		if _, isvalid := token.Method.(*jwt.SigningMethodHMAC); !isvalid {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])

		}  
		return []byte(secretKey), nil
	})

	if err != nil {
		log.Error("The refresh token not valid : %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error" : "renewRefreshToken",
		})
	}

	if claims, ok := token.Claims.(*auth.RefreshAuthCustomClaims); ok && token.Valid {
		if claims.Email ==  "" {
			log.Error("The email field received in the token is empty")
			c.Status(http.StatusBadRequest)
			return
		}
		userExists, err := database.UserExists(s.Database, claims.Email)
		if err != nil {
			log.Error("Error verifying if the user with email %s exists: %s", claims.Email, err)
			c.Status(http.StatusInternalServerError)
			return
		} else if userExists {
			user, err := database.GetUserDetailsForEmail(s.Database, claims.Email)
			if err != nil {
				log.Error("Error retrieving user details for email %s; %s", claims.Email, err)
				c.Status(http.StatusInternalServerError)
				return
			} else if !user.IsActivated {
				log.Error("The account %s is not activated", claims.Email)
				c.Status(http.StatusBadRequest)
				return
			} 
			newTokenPair := auth.JWTAuthService().GenerateToken(user.ID, user.Email, user.IsActivated)
			log.Info("Refresh token is valid. Successfully generated a new token pair")
			c.JSON(http.StatusOK, gin.H{
				"id" : user.ID,
				"refresh_token" : newTokenPair["refresh_token"],
				"token" : newTokenPair["access_token"],
			})
			return
		}

		log.Error("The user doesn't exist in the database")
		c.Status(http.StatusBadRequest)
		return
	}
}