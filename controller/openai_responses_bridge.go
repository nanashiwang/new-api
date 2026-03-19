package controller

import (
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func RelayResponsesMediaBridge(c *gin.Context) {
	service.ServeOpenAIResponsesMedia(c)
}
