package webserver

import (
	"net/http"

	"github.com/CosminMocanu97/dissertationBackend/internal/auth"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
	"github.com/gin-gonic/gin"
	cors "github.com/rs/cors/wrapper/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	LOGIN_CREDENTIALS_BINDING_ERROR = "Could not bind request data"
	CREDENTIALS_NOT_VERIFIED_ERROR  = "Could not finish credentials verification"
	INVALID_CREDENTIALS_ERROR       = "Invalid credentials"
	USER_ID_COULD_NOT_BE_RETRIEVED  = "Error retrieving the id for the user"
)

var (
	node = InitiateSnowflakeNode()
)

func Api(s *Service) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(cors.Default())

	// Routes
	// used for CORS
	r.GET("/", func(context *gin.Context) {
		context.Status(http.StatusOK)
	})
	r.GET("/ping", s.HandleGetPingRequest)
	r.POST("/mail", s.HandlePostMail)
	r.POST("/register", s.HandlePostRegisterRequest)
	r.POST("/login", s.HandlePostLoginRequest)
	r.GET("/activate/:token", s.HandlePostActivateAccount)
	r.POST("/forgot-password", s.HandlePostForgotPasswordRequest)
	r.POST("/renew-password/:token", s.HandlePostRenewPasswordRequest)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}

// JWT authorisation middleware

func AuthorizeJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		// if the token doesn't exist, return unauthorised
		if len(tokenString) == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "No authorization token in the request header",
			})
		}
		token, err := auth.JWTAuthService().ValidateToken(tokenString)
		if err != nil {
			log.Error("Error validating the JWT token %s: %s", tokenString, err)
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		if token.Valid {
			claims := token.Claims.(*auth.AuthCustomClaims)
			log.Info("token email: %s", claims.Email)
			c.Set("claims", claims)
			c.Next()
		} else {
			log.Error("%s", err)
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	}
}

//login contorller interface
type LoginController interface {
	Login(ctx *gin.Context) string
}

// handle GET "/ping" request
// HandleGetPingRequest godoc
// @Summary ping - pong
// @Description api health endpoint
// @Accept  json
// @Produce json
// @Success 200 {string} pong
// @Failure default {json} http.StatusInternalServerError
// @Router /ping [get]
func (s *Service) HandleGetPingRequest(c *gin.Context) {
	id := node.Generate()

	log.Info("UUID: %s; Request to GET /ping", id)
	c.String(http.StatusOK, "pong")
}