// Package reader serves the public tracking (kiosk) page and the theme state it
// polls to notice that an admin changed the page's appearance.
package reader

import (
	"net/http"

	themeview "github.com/buzyka/imlate/internal/usecase/theme"
	"github.com/gin-gonic/gin"
)

// readerTemplateName is the template registered on the gin engine at startup.
const readerTemplateName = "reader.html"

// ErrorResponse is the error payload for public reader endpoints.
type ErrorResponse struct {
	Error string `json:"error"`
}

type ReaderController struct {
	ThemeService *themeview.Service `container:"type"`
}

// ReaderPageHandler renders the tracking page. When the theme cannot be read the
// page still renders, using built-in defaults: a terminal that shows default
// artwork is far better than a terminal showing an error.
func (rc *ReaderController) ReaderPageHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := rc.ThemeService.GetReaderPageData()
		if err != nil {
			c.HTML(http.StatusOK, readerTemplateName, themeview.DefaultReaderPageData())
			return
		}
		c.HTML(http.StatusOK, readerTemplateName, data)
	}
}

// ThemeStateHandler godoc
// @Summary      Get tracking page theme state
// @Description  Returns the current tracking page theme together with a revision fingerprint.
// @Description  The tracking page polls this endpoint and refreshes its appearance only when the
// @Description  revision differs from the one it was rendered with, so an admin's theme change
// @Description  reaches long-running terminals without anyone reloading them by hand.
// @Description  Public by design: the payload is exactly what the public tracking page already
// @Description  renders, and terminals must be able to poll it before they are registered.
// @Tags         reader
// @Produce      json
// @Success      200  {object}  themeview.ReaderPageData
// @Failure      500  {object}  ErrorResponse
// @Router       /theme-state [get]
func (rc *ReaderController) ThemeStateHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")

		data, err := rc.ThemeService.GetReaderPageData()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, data)
	}
}
