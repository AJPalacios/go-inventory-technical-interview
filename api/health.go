package api

import (
	"time"

	"github.com/gin-gonic/gin"
)

func (server *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}
