package main

import (
	"net/http"

	"github.com/buzyka/imlate/internal/infrastructure/gocontainer"
	"github.com/buzyka/imlate/internal/isb/search"
	"github.com/buzyka/imlate/internal/isb/tracker"
	"github.com/gin-gonic/gin"
	"github.com/golobby/container/v3"
)

func main() {
	gocontainer.Build()
	r := gin.Default()

	r.Static("/assets", "./website/assets")

	// Define gita simple GET route
	r.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/", func(ctx *gin.Context) {
		ctx.File("website/index.html")
	})

	searchController := &search.SearchController{}
	container.MustFill(container.Global, searchController)
	r.GET("/search/:id", searchController.SearchHandler())
	
	trackerController := &tracker.TrackerController{}
	container.MustFill(container.Global, trackerController)
	r.POST("/track", trackerController.TrackHandler())

	// Start the server on port 8080
	r.Run(":8080")

}