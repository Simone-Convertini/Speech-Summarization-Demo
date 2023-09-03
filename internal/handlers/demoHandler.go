package handlers

import (
	"context"
	"net/http"

	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/repositories"
	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/services"
	"github.com/gin-gonic/gin"
)

type demoHandler struct {
	storeService   *services.StoreService
	scribeService  *services.ScribeService
	summaryService *services.SummaryService
}

func GetDemoHandler(ctx context.Context) *demoHandler {
	var sh demoHandler
	var sr repositories.StoreRepository

	sr.Context = ctx
	sh.storeService = services.GetStoreService(&sr)
	sh.scribeService = services.GetScribeService(&sr)
	sh.summaryService = services.GetSummaryService(&sr)
	return &sh
}

// Handlers:

func (sh *demoHandler) UploadFile(c *gin.Context) {

	file, err := c.FormFile("file") // "file" corresponds to the field name in the form
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	info, err := sh.storeService.UploadAudio(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": info})
}

func (sh *demoHandler) RunScriber() {
	sh.scribeService.MessageListenerRoutine()
}

func (sh *demoHandler) RunSummarizer() {
	sh.summaryService.MessageListenerRoutine()
}
