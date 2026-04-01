package adminapi

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/buzyka/imlate/internal/config"
	themeview "github.com/buzyka/imlate/internal/usecase/theme"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupThemeController(t *testing.T) *AdminAPIController {
	t.Helper()
	gin.SetMode(gin.TestMode)
	return &AdminAPIController{
		ThemeService: &themeview.Service{
			Config: &config.Config{
				ThemeDir:       t.TempDir(),
				ThemeURLPrefix: "/storage/theme",
			},
		},
	}
}

func TestGetThemeHandler_Defaults(t *testing.T) {
	controller := setupThemeController(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/theme", nil)

	controller.GetThemeHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"favicon\"")
	assert.Contains(t, w.Body.String(), "\"welcome_duration_ms\":1800")
}

func TestUploadThemeAssetHandler_Success(t *testing.T) {
	controller := setupThemeController(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "favicon.png")
	require.NoError(t, err)
	_, err = part.Write(createThemePNGBytes(t))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/theme/assets/favicon", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = gin.Params{{Key: "slot", Value: "favicon"}}

	controller.UploadThemeAssetHandler()(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"is_custom\":true")
	assert.Contains(t, w.Body.String(), "/storage/theme/favicon-")
}

func TestUpdateThemeSettingsHandler_Invalid(t *testing.T) {
	controller := setupThemeController(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body, err := json.Marshal(UpdateThemeSettingsRequest{})
	require.NoError(t, err)
	c.Request = httptest.NewRequest(http.MethodPut, "/admin-api/theme/settings", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.UpdateThemeSettingsHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetThemeAssetHandler_InvalidSlot(t *testing.T) {
	controller := setupThemeController(t)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/admin-api/theme/assets/unknown", nil)
	c.Params = gin.Params{{Key: "slot", Value: "unknown"}}

	controller.ResetThemeAssetHandler()(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func createThemePNGBytes(t *testing.T) []byte {
	t.Helper()
	src := image.NewRGBA(image.Rect(0, 0, 128, 128))
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			src.Set(x, y, color.RGBA{R: 200, G: 100, B: 80, A: 255})
		}
	}
	buf := bytes.NewBuffer(nil)
	require.NoError(t, png.Encode(buf, src))
	return buf.Bytes()
}
