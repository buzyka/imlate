package adminapi

import (
	"errors"
	"net/http"

	themeview "github.com/buzyka/imlate/internal/usecase/theme"
	"github.com/gin-gonic/gin"
)

const maxThemeAssetUploadSizeBytes int64 = 20 * 1024 * 1024

type UpdateThemeSettingsRequest struct {
	WelcomeDurationMs int `json:"welcome_duration_ms" binding:"required"`
	GoodbyeDurationMs int `json:"goodbye_duration_ms" binding:"required"`
}

func themeErrorStatus(err error) int {
	var themeErr *themeview.Error
	if ok := errors.As(err, &themeErr); ok {
		return themeErr.StatusCode
	}
	return http.StatusInternalServerError
}

// GetThemeHandler godoc
// @Summary      Get current theme
// @Description  Returns current tracking page theme assets and timing settings.
// @Tags         admin-theme
// @Produce      json
// @Success      200  {object}  themeview.Response
// @Failure      500  {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/theme [get]
func (ac *AdminAPIController) GetThemeHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := ac.ThemeService.GetTheme()
		if err != nil {
			c.JSON(themeErrorStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// UploadThemeAssetHandler godoc
// @Summary      Upload theme asset
// @Description  Uploads a custom asset for the given theme slot.
// @Tags         admin-theme
// @Accept       mpfd
// @Produce      json
// @Param        slot   path      string               true  "Theme slot"
// @Param        image  formData  file                 true  "Theme asset file"
// @Success      200    {object}  themeview.Response
// @Failure      400    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/theme/assets/{slot} [post]
func (ac *AdminAPIController) UploadThemeAssetHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		fileHeader, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "image file is required"})
			return
		}
		if fileHeader.Size > maxThemeAssetUploadSizeBytes {
			c.JSON(http.StatusBadRequest, gin.H{"error": "image file is too large (max 20MB)"})
			return
		}

		file, err := openUploadedFile(fileHeader)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
			return
		}
		defer func() { _ = file.Close() }()

		data, err := readUploadedFile(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
			return
		}

		resp, err := ac.ThemeService.UploadAsset(c.Param("slot"), fileHeader.Filename, data)
		if err != nil {
			c.JSON(themeErrorStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// ResetThemeAssetHandler godoc
// @Summary      Reset theme asset
// @Description  Removes a custom asset assignment and restores the default slot asset.
// @Tags         admin-theme
// @Produce      json
// @Param        slot  path      string               true  "Theme slot"
// @Success      200   {object}  themeview.Response
// @Failure      400   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/theme/assets/{slot} [delete]
func (ac *AdminAPIController) ResetThemeAssetHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := ac.ThemeService.ResetAsset(c.Param("slot"))
		if err != nil {
			c.JSON(themeErrorStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// UpdateThemeSettingsHandler godoc
// @Summary      Update theme settings
// @Description  Updates welcome and goodbye animation durations for the tracking page.
// @Tags         admin-theme
// @Accept       json
// @Produce      json
// @Param        request  body      UpdateThemeSettingsRequest  true  "Theme settings request"
// @Success      200      {object}  themeview.Response
// @Failure      400      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/theme/settings [put]
func (ac *AdminAPIController) UpdateThemeSettingsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req UpdateThemeSettingsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := ac.ThemeService.UpdateSettings(themeview.Settings{
			WelcomeDurationMs: req.WelcomeDurationMs,
			GoodbyeDurationMs: req.GoodbyeDurationMs,
		})
		if err != nil {
			c.JSON(themeErrorStatus(err), gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}
