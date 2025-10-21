package main

import (
	"fmt"
	"net/http"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/infrastructure/gocontainer"
	"github.com/buzyka/imlate/internal/infrastructure/util"
	"github.com/buzyka/imlate/internal/isb/search"
	"github.com/buzyka/imlate/internal/isb/tracker"
	"github.com/buzyka/imlate/internal/isb/visitor"
	"github.com/gin-gonic/gin"
	"github.com/golobby/container/v3"
	"github.com/subosito/gotenv"
)

func main() {
	envFiles := []string{".env"}
	if rootPath, err := util.GetRootPath(); err == nil {
		if util.FileExists(rootPath + "/.env") {
			envFiles = append(envFiles, rootPath + "/.env")
		}
	}
	_ = gotenv.Load(envFiles...)

	cfg, err := config.NewFromEnv()
	if err != nil {
		panic(fmt.Sprintf("Error loading config from env: %v\n", err))
	}
	
	gocontainer.Build(&cfg)
	r := gin.Default()

	r.Static("/assets", "./website/assets")
	r.Static("/output", "./output")

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

	r.GET("/add", func(ctx *gin.Context) {
		ctx.File("website/add-key.html")
	})

	searchController := &search.SearchController{}
	container.MustFill(container.Global, searchController)
	r.GET("/search/:id", searchController.SearchHandler())
	
	apiRouteGroup := r.Group("/api")
	trackerController := &tracker.TrackerController{}
	container.MustFill(container.Global, trackerController)
	r.POST("/change-time", trackerController.ChangeTimeHandler())
	r.POST("/track", trackerController.TrackHandler())
	r.POST("/find-and-track", trackerController.FindAndTrackHandler())
	visitorController := &visitor.VisitorController{}
	container.MustFill(container.Global, visitorController)
	apiRouteGroup.PATCH("/add-key", visitorController.AddKeyHandler())

	// Start the server on port 8080
	r.Run("0.0.0.0:8080")
}
