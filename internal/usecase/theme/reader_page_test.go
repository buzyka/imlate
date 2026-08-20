package theme

import (
	"testing"

	"github.com/buzyka/imlate/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultReaderPageData(t *testing.T) {
	data := DefaultReaderPageData()

	assert.Equal(t, DefaultFaviconURL, data.FaviconURL)
	assert.Equal(t, DefaultLogoBackgroundURL, data.LogoBackgroundURL)
	assert.Equal(t, DefaultWelcomeAnimationURL, data.WelcomeAnimationURL)
	assert.Equal(t, DefaultGoodbyeAnimationURL, data.GoodbyeAnimationURL)
	assert.Equal(t, DefaultWelcomeDurationMs, data.WelcomeDurationMs)
	assert.Equal(t, DefaultGoodbyeDurationMs, data.GoodbyeDurationMs)

	assert.Len(t, data.Revision, readerRevisionLength)
	assert.Equal(t, version.Version, data.AppVersion)
	assert.Equal(t, DefaultReaderThemePollSeconds, data.PollSeconds)
}

// The page and the endpoint must agree on what "unchanged" means, so an untouched
// theme has to produce the same revision every time it is read.
func TestGetReaderPageData_RevisionStableWhenNothingChanged(t *testing.T) {
	service := newTestThemeService(t)

	first, err := service.GetReaderPageData()
	require.NoError(t, err)
	second, err := service.GetReaderPageData()
	require.NoError(t, err)

	assert.Equal(t, first.Revision, second.Revision)
	assert.Equal(t, DefaultReaderPageData().Revision, first.Revision)
}

// Rewriting the manifest with identical values must not look like a change:
// hashing the file or its mtime instead of the rendered values would report one.
func TestGetReaderPageData_RevisionUnchangedByIdenticalManifestRewrite(t *testing.T) {
	service := newTestThemeService(t)

	before, err := service.GetReaderPageData()
	require.NoError(t, err)

	_, err = service.UpdateSettings(Settings{
		WelcomeDurationMs: DefaultWelcomeDurationMs,
		GoodbyeDurationMs: DefaultGoodbyeDurationMs,
	})
	require.NoError(t, err)

	after, err := service.GetReaderPageData()
	require.NoError(t, err)
	assert.Equal(t, before.Revision, after.Revision)
}

func TestGetReaderPageData_RevisionChangesOnSettingsUpdate(t *testing.T) {
	service := newTestThemeService(t)

	before, err := service.GetReaderPageData()
	require.NoError(t, err)

	_, err = service.UpdateSettings(Settings{WelcomeDurationMs: 2500, GoodbyeDurationMs: 3200})
	require.NoError(t, err)

	after, err := service.GetReaderPageData()
	require.NoError(t, err)
	assert.NotEqual(t, before.Revision, after.Revision)
	assert.Equal(t, 2500, after.WelcomeDurationMs)
	assert.Equal(t, 3200, after.GoodbyeDurationMs)
}

func TestGetReaderPageData_RevisionChangesOnAssetUploadAndReset(t *testing.T) {
	service := newTestThemeService(t)

	initial, err := service.GetReaderPageData()
	require.NoError(t, err)

	_, err = service.UploadAsset("logo_background", "logo.png", createPNGBytes(t, 400, 300))
	require.NoError(t, err)

	uploaded, err := service.GetReaderPageData()
	require.NoError(t, err)
	assert.NotEqual(t, initial.Revision, uploaded.Revision)
	assert.Contains(t, uploaded.LogoBackgroundURL, "/storage/theme/logo_background-")

	_, err = service.ResetAsset("logo_background")
	require.NoError(t, err)

	reset, err := service.GetReaderPageData()
	require.NoError(t, err)
	assert.NotEqual(t, uploaded.Revision, reset.Revision)
	// Back to the default artwork, so back to the original revision.
	assert.Equal(t, initial.Revision, reset.Revision)
}

// Each of the six rendered fields has to be part of the fingerprint; a field left
// out would make a real theme change invisible to every terminal.
func TestReaderRevision_CoversEveryRenderedField(t *testing.T) {
	base := ReaderPageData{
		FaviconURL:          "/a.ico",
		LogoBackgroundURL:   "/b.png",
		WelcomeAnimationURL: "/c.gif",
		GoodbyeAnimationURL: "/d.gif",
		WelcomeDurationMs:   1800,
		GoodbyeDurationMs:   1900,
	}
	baseRevision := readerRevision(base)

	mutations := map[string]func(*ReaderPageData){
		"favicon":       func(d *ReaderPageData) { d.FaviconURL = "/changed.ico" },
		"logo":          func(d *ReaderPageData) { d.LogoBackgroundURL = "/changed.png" },
		"welcome_asset": func(d *ReaderPageData) { d.WelcomeAnimationURL = "/changed.gif" },
		"goodbye_asset": func(d *ReaderPageData) { d.GoodbyeAnimationURL = "/changed.gif" },
		"welcome_ms":    func(d *ReaderPageData) { d.WelcomeDurationMs = 2000 },
		"goodbye_ms":    func(d *ReaderPageData) { d.GoodbyeDurationMs = 2000 },
	}

	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := base
			mutate(&changed)
			assert.NotEqual(t, baseRevision, readerRevision(changed))
		})
	}
}

// Revision must not depend on the deploy-specific fields, otherwise a restart
// would look like a theme change to every terminal.
func TestReaderRevision_IgnoresDerivedFields(t *testing.T) {
	base := ReaderPageData{FaviconURL: "/a.ico", WelcomeDurationMs: 1800}
	withDerived := base
	withDerived.Revision = "stale"
	withDerived.AppVersion = "9.9.9"
	withDerived.PollSeconds = 42

	assert.Equal(t, readerRevision(base), readerRevision(withDerived))
}

// The fingerprint must not be confusable by shifting content between fields.
func TestReaderRevision_FieldsAreDelimited(t *testing.T) {
	a := ReaderPageData{FaviconURL: "/a", LogoBackgroundURL: "/b"}
	b := ReaderPageData{FaviconURL: "/a\n/b"}

	assert.NotEqual(t, readerRevision(a), readerRevision(b))
}

func TestGetReaderPageData_PollSecondsFromConfig(t *testing.T) {
	service := newTestThemeService(t)
	service.Config.ReaderThemePollSeconds = 60

	data, err := service.GetReaderPageData()

	require.NoError(t, err)
	assert.Equal(t, 60, data.PollSeconds)
}

// A misconfigured interval must not turn every terminal into a request generator.
func TestNormalizePollSeconds(t *testing.T) {
	testCases := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero falls back to default", 0, DefaultReaderThemePollSeconds},
		{"negative falls back to default", -5, DefaultReaderThemePollSeconds},
		{"below floor is raised", 1, minReaderThemePollSeconds},
		{"floor is kept", minReaderThemePollSeconds, minReaderThemePollSeconds},
		{"normal value is kept", 900, 900},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, normalizePollSeconds(tc.input))
		})
	}
}

func TestGetReaderPageData_ManifestError(t *testing.T) {
	service := newTestThemeService(t)
	writeThemeManifest(t, service.Config.ThemeDir, "{ not json")

	_, err := service.GetReaderPageData()

	assert.Error(t, err)
}
