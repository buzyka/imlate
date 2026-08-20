package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readReaderTemplate(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "website", "reader.html"))
	require.NoError(t, err)
	return string(data)
}

func TestReaderTemplate_ShowsFullLogoWithoutBackgroundCropping(t *testing.T) {
	template := readReaderTemplate(t)

	assert.Contains(t, template, `<img id="logo-image"`)
	assert.Contains(t, template, `object-fit: contain`)
	assert.NotContains(t, template, `background-size: cover`)
	assert.False(t, strings.Contains(template, `logoBlock.style.backgroundImage = "url('`))
}

// The kiosk page never reloads on its own, so without this poll an admin's theme
// change would never reach a running terminal.
func TestReaderTemplate_PollsThemeState(t *testing.T) {
	template := readReaderTemplate(t)

	assert.Contains(t, template, `xhr.open('GET', '/theme-state', true)`)
	assert.Contains(t, template, `setInterval(pollThemeState, themePollMs)`)
	assert.Contains(t, template, `var themePollMs         = {{ .PollSeconds }} * 1000;`)
	assert.Contains(t, template, `var themeRevision       = '{{ .Revision }}';`)
	assert.Contains(t, template, `var appVersion          = '{{ .AppVersion }}';`)
}

// Animation URLs have to live in variables the poller can reassign; rendering them
// straight into the assignment would make the theme unpatchable.
func TestReaderTemplate_AnimationUrlsAreReassignable(t *testing.T) {
	template := readReaderTemplate(t)

	assert.Contains(t, template, `var welcomeAnimationUrl = '{{ .WelcomeAnimationURL }}';`)
	assert.Contains(t, template, `var goodbyeAnimationUrl = '{{ .GoodbyeAnimationURL }}';`)
	assert.Contains(t, template, `welcomeIcon.src = welcomeAnimationUrl;`)
	assert.Contains(t, template, `welcomeIcon.src = goodbyeAnimationUrl;`)
	assert.NotContains(t, template, `welcomeIcon.src = '{{ .WelcomeAnimationURL }}'`)
	assert.NotContains(t, template, `welcomeIcon.src = '{{ .GoodbyeAnimationURL }}'`)
}

// A theme swap must wait for the terminal to go idle: replacing the animation
// image mid-playback is a visible glitch, and a reload during registration would
// discard whatever the admin was typing.
func TestReaderTemplate_DefersThemeUpdatesUntilIdle(t *testing.T) {
	template := readReaderTemplate(t)

	// An in-place patch waits only for an animation to finish. Gating it on the
	// registration modal too would leave an unregistered terminal — where that modal
	// stays open until someone registers it — stuck on the old theme forever.
	assert.Regexp(t, `function canApplyThemeInPlace\(\) \{\s*return !animationInProgress;`, template)
	assert.Regexp(t, `function canReloadPage\(\) \{\s*return !regModalOpen && !animationInProgress;`, template)
	assert.Contains(t, template, `animationInProgress = true;`)
	// Both transitions back to idle must flush the pending state, otherwise a theme
	// change deferred during a scan would sit unapplied until the next poll.
	assert.Regexp(t, `animationInProgress = false;\s*applyPendingThemeState\(\);`, template)
	assert.Regexp(t, `rfidInput\.focus\(\);\s*applyPendingThemeState\(\);`, template)
}

// A full reload is reserved for a version change: the page loads jQuery, Popper and
// Bootstrap from external CDNs, so reloading while offline would break it for good.
func TestReaderTemplate_ReloadsOnlyOnAppVersionChange(t *testing.T) {
	template := readReaderTemplate(t)

	assert.Equal(t, 1, strings.Count(template, `location.reload()`))
	// The single reload must sit behind both the app-version check and the
	// terminal-is-free check, in that order.
	assert.Regexp(t,
		`if \(pendingThemeState\.app_version !== appVersion\) \{\s*if \(!canReloadPage\(\)\) return;\s*location\.reload\(\);`,
		template)
}
