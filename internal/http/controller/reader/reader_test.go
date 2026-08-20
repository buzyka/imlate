package reader

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/buzyka/imlate/internal/config"
	themeview "github.com/buzyka/imlate/internal/usecase/theme"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func newTestController(t *testing.T) *ReaderController {
	t.Helper()
	return &ReaderController{
		ThemeService: &themeview.Service{
			Config: &config.Config{
				ThemeDir:               filepath.Join(t.TempDir(), "theme"),
				ThemeURLPrefix:         "/storage/theme",
				ReaderThemePollSeconds: 300,
			},
		},
	}
}

// breakThemeManifest makes the theme unreadable so error paths can be exercised.
func breakThemeManifest(t *testing.T, themeDir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "theme.json"), []byte("{ not json"), 0o644))
}

func performRequest(handler gin.HandlerFunc, path string) *httptest.ResponseRecorder {
	router := gin.New()
	router.SetHTMLTemplate(template.Must(template.New("reader.html").Parse(
		`revision={{ .Revision }} logo={{ .LogoBackgroundURL }} poll={{ .PollSeconds }}`,
	)))
	router.GET(path, handler)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestThemeStateHandler_Success(t *testing.T) {
	controller := newTestController(t)

	w := performRequest(controller.ThemeStateHandler(), "/theme-state")

	require.Equal(t, http.StatusOK, w.Code)

	var body themeview.ReaderPageData
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, themeview.DefaultLogoBackgroundURL, body.LogoBackgroundURL)
	assert.Equal(t, themeview.DefaultWelcomeDurationMs, body.WelcomeDurationMs)
	assert.NotEmpty(t, body.Revision)
	assert.NotEmpty(t, body.AppVersion)
	assert.Equal(t, 300, body.PollSeconds)
}

// A cached response would hide the very change the page is polling for.
func TestThemeStateHandler_IsNotCacheable(t *testing.T) {
	controller := newTestController(t)

	w := performRequest(controller.ThemeStateHandler(), "/theme-state")

	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

// The JSON payload keys are the contract the tracking page reads.
func TestThemeStateHandler_PayloadKeys(t *testing.T) {
	controller := newTestController(t)

	w := performRequest(controller.ThemeStateHandler(), "/theme-state")

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	for _, key := range []string{
		"favicon_url", "logo_background_url", "welcome_animation_url",
		"goodbye_animation_url", "welcome_duration_ms", "goodbye_duration_ms",
		"revision", "app_version", "poll_seconds",
	} {
		assert.Contains(t, body, key)
	}
}

// On failure the endpoint must not report default URLs: that would tell every
// terminal the theme had been reset and make it swap to default artwork, then
// swap back on the next successful poll.
func TestThemeStateHandler_ErrorDoesNotReportDefaults(t *testing.T) {
	controller := newTestController(t)
	breakThemeManifest(t, controller.ThemeService.Config.ThemeDir)

	w := performRequest(controller.ThemeStateHandler(), "/theme-state")

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), themeview.DefaultLogoBackgroundURL)
	assert.NotContains(t, w.Body.String(), "revision")

	var body ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.NotEmpty(t, body.Error)
}

func TestReaderPageHandler_Success(t *testing.T) {
	controller := newTestController(t)

	w := performRequest(controller.ReaderPageHandler(), "/")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "logo="+themeview.DefaultLogoBackgroundURL)
	assert.Contains(t, w.Body.String(), "poll=300")
	assert.NotContains(t, w.Body.String(), "revision= ")
}

// Unlike the state endpoint, the page must still render when the theme cannot be
// read: a terminal showing default artwork beats a terminal showing an error.
func TestReaderPageHandler_FallsBackToDefaultsOnError(t *testing.T) {
	controller := newTestController(t)
	breakThemeManifest(t, controller.ThemeService.Config.ThemeDir)

	w := performRequest(controller.ReaderPageHandler(), "/")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "logo="+themeview.DefaultLogoBackgroundURL)
	assert.Contains(t, w.Body.String(), "revision="+themeview.DefaultReaderPageData().Revision)
}

// The page bakes in the revision it was rendered with and compares it against the
// endpoint's; if the two disagreed while nothing had changed, every terminal would
// refresh its theme on the first poll.
func TestReaderPageAndThemeStateAgreeOnRevision(t *testing.T) {
	controller := newTestController(t)

	page := performRequest(controller.ReaderPageHandler(), "/")
	state := performRequest(controller.ThemeStateHandler(), "/theme-state")

	var body themeview.ReaderPageData
	require.NoError(t, json.Unmarshal(state.Body.Bytes(), &body))
	assert.Contains(t, page.Body.String(), "revision="+body.Revision)
}
