package main

import (
	"fmt"
	"net/http"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/infrastructure/gocontainer"
	"github.com/buzyka/imlate/internal/infrastructure/util"
	"github.com/buzyka/imlate/internal/isb/search"
	"github.com/buzyka/imlate/internal/isb/tracker"
	"github.com/gin-gonic/gin"
	"github.com/golobby/container/v3"
	"github.com/subosito/gotenv"
)

func main() {
	if rootPath, err := util.GetRootPath(); err == nil {
		if util.FileExists(rootPath + "/.env") {
			_ = gotenv.Load(rootPath + "/.env")
		}
	}

	cfg, err := config.NewFromEnv()
	if err != nil {
		panic(fmt.Sprintf("Error loading config from env: %v\n", err))
	}
	
	gocontainer.Build(&cfg)
	r := gin.Default()

	r.Static("/assets", "./website/assets")

	// Define gita simple GET route
	r.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/", func(ctx *gin.Context) {
		ctx.File("website/reader.html")
	})

	r.GET("/manual", func(ctx *gin.Context) {
		ctx.File("website/index.html")
	})

	searchController := &search.SearchController{}
	container.MustFill(container.Global, searchController)
	r.GET("/search/:id", searchController.SearchHandler())
	
	trackerController := &tracker.TrackerController{}
	container.MustFill(container.Global, trackerController)
	r.POST("/track", trackerController.TrackHandler())
	r.POST("/find-and-track", trackerController.FindAndTrackHandler())

	// Start the server on port 8080
	r.Run("0.0.0.0:8080")
}