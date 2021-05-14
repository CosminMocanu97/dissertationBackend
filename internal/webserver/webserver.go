package webserver

import (
	"net/http"

	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
	cors "github.com/rs/cors/wrapper/gin"
)

func InitiateSnowflakeNode() *snowflake.Node {
	node, err := snowflake.NewNode(1)
	if err != nil {
		log.Fatal("Error initializing snowflake %s", err)
	}

	return node
}

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
	r.GET("/test", s.HandleGetTestRequest)
	return r
}

func (s *Service) HandleGetPingRequest(c *gin.Context) {
	id := node.Generate()

	log.Info("UUID: %s; Request to GET /ping", id)
	c.String(http.StatusOK, "pong")
}

func (s *Service) HandleGetTestRequest(c *gin.Context) {
	id := node.Generate()

	log.Info("UUID: %s; Request to GET /ping", id)
	c.JSON(http.StatusOK, gin.H{
		"something": "test",
	})
}