package theme

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strings"
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

// The uploaded file name is only a label from the multipart body and has no say
// in where the asset lands: whatever it looks like, the file is written inside
// ThemeDir under a slot-prefixed name, with the extension taken from the
// detected content type.
func TestServiceUploadAsset_OddFilenameStillWritesInsideThemeDir(t *testing.T) {
	filenames := []string{
		"../../../../etc/cron.d/script.sh",
		"..%2f..%2fimage.png",
		"favicon.png/../../../../tmp/image.png",
		`..\..\windows\system32\image.png`,
		"image.png\x00.txt",
		strings.Repeat("a", 300) + ".png",
		"",
	}

	for _, filename := range filenames {
		service, rootDir := newTestThemeServiceWithParentDir(t)

		resp, err := service.UploadAsset("favicon", filename, createPNGBytes(t, 64, 64))
		require.NoError(t, err, "filename %q", filename)
		assert.True(t, resp.Assets["favicon"].IsCustom)

		files, err := os.ReadDir(service.Config.ThemeDir)
		require.NoError(t, err)

		var written []string
		for _, f := range files {
			if f.Name() != "theme.json" {
				written = append(written, f.Name())
			}
		}
		require.Len(t, written, 1, "filename %q", filename)
		assert.True(t, strings.HasPrefix(written[0], "favicon-"), "filename %q wrote %q", filename, written[0])
		assert.Equal(t, ".png", filepath.Ext(written[0]), "filename %q wrote %q", filename, written[0])

		// The directory above ThemeDir is left untouched.
		parentEntries, err := os.ReadDir(rootDir)
		require.NoError(t, err)
		require.Len(t, parentEntries, 1, "filename %q also wrote into %s", filename, rootDir)
		assert.Equal(t, "theme", parentEntries[0].Name())
	}
}

// themeAssetPath is the single place that turns an asset name into a path, so
// it accepts plain file names only.
func TestThemeAssetPath_RejectsNonPlainNames(t *testing.T) {
	service := newTestThemeService(t)

	for _, name := range []string{"", ".", "..", "../other", "sub/dir.png", `back\slash.png`} {
		_, err := service.themeAssetPath(name)
		assert.Error(t, err, "name %q", name)
	}

	path, err := service.themeAssetPath("favicon-123.png")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(service.Config.ThemeDir, "favicon-123.png"), path)
}
