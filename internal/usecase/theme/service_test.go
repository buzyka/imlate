package theme

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceGetTheme_Defaults(t *testing.T) {
	service := newTestThemeService(t)

	resp, err := service.GetTheme()

	require.NoError(t, err)
	assert.Equal(t, DefaultFaviconURL, resp.Assets["favicon"].CurrentURL)
	assert.Equal(t, DefaultLogoBackgroundURL, resp.Assets["logo_background"].CurrentURL)
	assert.False(t, resp.Assets["favicon"].IsCustom)
	assert.Equal(t, 1800, resp.Settings.WelcomeDurationMs)
	assert.Equal(t, 1800, resp.Settings.GoodbyeDurationMs)
}

func TestServiceUploadResetAndSettings(t *testing.T) {
	service := newTestThemeService(t)

	uploaded, err := service.UploadAsset("favicon", "favicon.png", createPNGBytes(t, 512, 512))
	require.NoError(t, err)
	assert.True(t, uploaded.Assets["favicon"].IsCustom)
	assert.Contains(t, uploaded.Assets["favicon"].CurrentURL, "/storage/theme/favicon-")

	files, err := os.ReadDir(service.Config.ThemeDir)
	require.NoError(t, err)
	assert.Len(t, files, 2)

	updated, err := service.UpdateSettings(Settings{
		WelcomeDurationMs: 2500,
		GoodbyeDurationMs: 3200,
	})
	require.NoError(t, err)
	assert.Equal(t, 2500, updated.Settings.WelcomeDurationMs)
	assert.Equal(t, 3200, updated.Settings.GoodbyeDurationMs)

	reset, err := service.ResetAsset("favicon")
	require.NoError(t, err)
	assert.False(t, reset.Assets["favicon"].IsCustom)
	assert.Equal(t, DefaultFaviconURL, reset.Assets["favicon"].CurrentURL)

	manifestData, err := os.ReadFile(filepath.Join(service.Config.ThemeDir, "theme.json"))
	require.NoError(t, err)
	assert.Contains(t, string(manifestData), "\"welcome_duration_ms\": 2500")
	assert.NotContains(t, string(manifestData), "\"favicon\"")
}

func TestServiceUploadAsset_InvalidSlot(t *testing.T) {
	service := newTestThemeService(t)

	resp, err := service.UploadAsset("unknown", "a.png", []byte("data"))

	require.Error(t, err)
	assert.Nil(t, resp)
	var themeErr *Error
	assert.ErrorAs(t, err, &themeErr)
	assert.Equal(t, 400, themeErr.StatusCode)
}

func TestServiceUploadAsset_GIFPreserved(t *testing.T) {
	service := newTestThemeService(t)

	resp, err := service.UploadAsset("welcome_animation", "welcome.gif", createGIFBytes(t))

	require.NoError(t, err)
	assert.True(t, resp.Assets["welcome_animation"].IsCustom)

	files, err := os.ReadDir(service.Config.ThemeDir)
	require.NoError(t, err)
	var gifName string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".gif" {
			gifName = file.Name()
			break
		}
	}
	require.NotEmpty(t, gifName)
	data, err := os.ReadFile(filepath.Join(service.Config.ThemeDir, gifName))
	require.NoError(t, err)
	assert.Equal(t, createGIFBytes(t), data)
}

func TestServiceUploadAsset_RejectsWebPLogoBackground(t *testing.T) {
	service := newTestThemeService(t)

	resp, err := service.UploadAsset("logo_background", "background.webp", fakeWebPBytes())

	require.Error(t, err)
	assert.Nil(t, resp)
	var themeErr *Error
	assert.ErrorAs(t, err, &themeErr)
	assert.Equal(t, 400, themeErr.StatusCode)
	assert.Contains(t, themeErr.Message, "unsupported image type")
}

func TestServiceUpdateSettings_Invalid(t *testing.T) {
	service := newTestThemeService(t)

	resp, err := service.UpdateSettings(Settings{})

	require.Error(t, err)
	assert.Nil(t, resp)
}

func TestServiceGetTheme_IgnoresInvalidManifestFileName(t *testing.T) {
	service, _ := newTestThemeServiceWithParentDir(t)
	writeThemeManifest(t, service.Config.ThemeDir, `{
  "assets": {
    "favicon": {
      "file_name": "../outside.png",
      "content_type": "image/png",
      "size_bytes": 123
    }
  }
}`)

	resp, err := service.GetTheme()

	require.NoError(t, err)
	assert.False(t, resp.Assets["favicon"].IsCustom)
	assert.Equal(t, DefaultFaviconURL, resp.Assets["favicon"].CurrentURL)
}

func TestServiceResetAsset_DoesNotDeleteOutsideThemeDir(t *testing.T) {
	service, rootDir := newTestThemeServiceWithParentDir(t)
	victimPath := filepath.Join(rootDir, "victim.txt")
	require.NoError(t, os.WriteFile(victimPath, []byte("keep"), 0o644))
	writeThemeManifest(t, service.Config.ThemeDir, `{
  "assets": {
    "favicon": {
      "file_name": "../victim.txt",
      "content_type": "image/png",
      "size_bytes": 4
    }
  }
}`)

	_, err := service.ResetAsset("favicon")

	require.NoError(t, err)
	_, statErr := os.Stat(victimPath)
	require.NoError(t, statErr)
}

func TestServiceUploadAsset_DoesNotDeleteOutsideThemeDirOnReplace(t *testing.T) {
	service, rootDir := newTestThemeServiceWithParentDir(t)
	victimPath := filepath.Join(rootDir, "victim.txt")
	require.NoError(t, os.WriteFile(victimPath, []byte("keep"), 0o644))
	writeThemeManifest(t, service.Config.ThemeDir, `{
  "assets": {
    "favicon": {
      "file_name": "../victim.txt",
      "content_type": "image/png",
      "size_bytes": 4
    }
  }
}`)

	resp, err := service.UploadAsset("favicon", "favicon.png", createPNGBytes(t, 64, 64))

	require.NoError(t, err)
	assert.True(t, resp.Assets["favicon"].IsCustom)
	_, statErr := os.Stat(victimPath)
	require.NoError(t, statErr)
}

func TestServiceUploadAsset_ReplacesPreviousThemeFile(t *testing.T) {
	service := newTestThemeService(t)

	firstResp, err := service.UploadAsset("favicon", "favicon.png", createPNGBytes(t, 64, 64))
	require.NoError(t, err)
	firstFileName := filepath.Base(firstResp.Assets["favicon"].CurrentURL)
	firstPath := filepath.Join(service.Config.ThemeDir, firstFileName)
	_, statErr := os.Stat(firstPath)
	require.NoError(t, statErr)

	service.now = func() time.Time {
		return time.Date(2026, 3, 31, 12, 0, 1, 0, time.UTC)
	}

	secondResp, err := service.UploadAsset("favicon", "favicon.png", createPNGBytes(t, 32, 32))
	require.NoError(t, err)
	secondFileName := filepath.Base(secondResp.Assets["favicon"].CurrentURL)
	secondPath := filepath.Join(service.Config.ThemeDir, secondFileName)

	assert.NotEqual(t, firstFileName, secondFileName)
	_, statErr = os.Stat(firstPath)
	require.ErrorIs(t, statErr, os.ErrNotExist)
	_, statErr = os.Stat(secondPath)
	require.NoError(t, statErr)
}

func newTestThemeService(t *testing.T) *Service {
	t.Helper()
	service, _ := newTestThemeServiceWithParentDir(t)
	return service
}

func newTestThemeServiceWithParentDir(t *testing.T) (*Service, string) {
	t.Helper()
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	rootDir := t.TempDir()
	themeDir := filepath.Join(rootDir, "theme")
	return &Service{
		Config: &config.Config{
			ThemeDir:       themeDir,
			ThemeURLPrefix: "/storage/theme",
		},
		now: func() time.Time { return now },
	}, rootDir
}

func createPNGBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	src := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			src.Set(x, y, color.RGBA{R: 100, G: 120, B: 200, A: 255})
		}
	}
	buf := bytes.NewBuffer(nil)
	require.NoError(t, png.Encode(buf, src))
	return buf.Bytes()
}

func createGIFBytes(t *testing.T) []byte {
	t.Helper()
	palette := color.Palette{color.Black, color.White}
	frame := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	frame.Pix = []uint8{0, 1, 1, 0}
	buf := bytes.NewBuffer(nil)
	require.NoError(t, gif.EncodeAll(buf, &gif.GIF{
		Image: []*image.Paletted{frame},
		Delay: []int{10},
	}))
	return buf.Bytes()
}

func writeThemeManifest(t *testing.T, themeDir, contents string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "theme.json"), []byte(contents), 0o644))
}

func fakeWebPBytes() []byte {
	return []byte{
		'R', 'I', 'F', 'F',
		0x1a, 0x00, 0x00, 0x00,
		'W', 'E', 'B', 'P',
		'V', 'P', '8', ' ',
		0x0e, 0x00, 0x00, 0x00,
		0x2f, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
}
