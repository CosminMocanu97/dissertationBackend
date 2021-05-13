package webserver

import (
	"github.com/gin-gonic/gin"
	"github.com/CosminMocanu97/dissertationBackend/internal/types"
	"github.com/CosminMocanu97/dissertationBackend/pkg/log"
	"net/http"
)

// handle POST "/mail" request
func (s *Service) HandlePostMail(c *gin.Context) {
	// generate ID for the request
	id := node.Generate()

	var email types.Email
	err := c.BindJSON(&email)
	if err != nil {
		log.Error("UUID: %s; Error: %s binding the json for the request %s with id %s", id, err.Error(), c.Request.Body, id)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err,
		})
	}

	err = s.MailingService.SendEmail(email.Recipients, email.Subject, email.Payload, "", id)
	if err != nil {
		log.Error("UUID: %s; Error %s sending the email", id, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
	}
	c.Status(http.StatusOK)
}