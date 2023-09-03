package main

import (
	"context"
	"fmt"

	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/handlers"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load("internal/config/.env")
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	ctx := context.Background()
	demoHandler := handlers.GetDemoHandler(ctx)

	// Events Handling Routines
	go demoHandler.RunScriber()
	go demoHandler.RunSummarizer()

	// GIN
	app := gin.Default()
	app.POST("/stores", demoHandler.UploadFile)
	app.Run()
}
