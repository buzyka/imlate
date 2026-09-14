package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func websitePath(parts ...string) string {
	return filepath.Join(append([]string{"..", "..", "website"}, parts...)...)
}

func readWebsiteFile(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(websitePath(parts...))
	require.NoError(t, err)
	return string(data)
}

func readReaderTemplate(t *testing.T) string {
	t.Helper()
	return readWebsiteFile(t, "reader.html")
}

func readReaderStyles(t *testing.T) string {
	t.Helper()
	return readWebsiteFile(t, "assets", "css", "reader.css")
}

func readReaderScript(t *testing.T) string {
	t.Helper()
	return readWebsiteFile(t, "assets", "js", "reader.js")
}

// readReaderPage returns every source the kiosk page is assembled from, joined
// in load order. Invariants about the page as a whole — "exactly one
// location.reload() anywhere" — are asserted against this, so moving code
// between the template, the stylesheet and the script can never silently
// reintroduce a second copy in the file the test stopped looking at.
func readReaderPage(t *testing.T) string {
	t.Helper()
	return strings.Join([]string{
		readReaderTemplate(t),
		readReaderStyles(t),
		readReaderScript(t),
	}, "\n")
}

func TestReaderTemplate_ShowsFullLogoWithoutBackgroundCropping(t *testing.T) {
	html := readReaderTemplate(t)
	css := readReaderStyles(t)
	page := readReaderPage(t)

	assert.Contains(t, html, `<img id="logo-image"`)
	assert.Contains(t, css, `object-fit: contain`)
	assert.NotContains(t, page, `background-size: cover`)
	assert.NotContains(t, page, `logoBlock.style.backgroundImage = "url('`)
}

// The kiosk page never reloads on its own, so without this poll an admin's theme
// change would never reach a running terminal.
func TestReaderTemplate_PollsThemeState(t *testing.T) {
	html := readReaderTemplate(t)
	js := readReaderScript(t)

	assert.Contains(t, js, `xhr.open('GET', '/theme-state', true)`)
	assert.Contains(t, js, `setInterval(pollThemeState, themePollMs)`)
	assert.Contains(t, html, `var themePollMs         = {{ .PollSeconds }} * 1000;`)
	assert.Contains(t, html, `var themeRevision       = '{{ .Revision }}';`)
	assert.Contains(t, html, `var appVersion          = '{{ .AppVersion }}';`)
}

// Animation URLs have to live in variables the poller can reassign; rendering them
// straight into the assignment would make the theme unpatchable.
func TestReaderTemplate_AnimationUrlsAreReassignable(t *testing.T) {
	html := readReaderTemplate(t)
	js := readReaderScript(t)
	page := readReaderPage(t)

	assert.Contains(t, html, `var welcomeAnimationUrl = '{{ .WelcomeAnimationURL }}';`)
	assert.Contains(t, html, `var goodbyeAnimationUrl = '{{ .GoodbyeAnimationURL }}';`)
	assert.Contains(t, js, `welcomeIcon.src = welcomeAnimationUrl;`)
	assert.Contains(t, js, `welcomeIcon.src = goodbyeAnimationUrl;`)
	// A template action can only be interpolated in reader.html, but assert on the
	// whole page so an inline <script> that regrows is caught too.
	assert.NotContains(t, page, `welcomeIcon.src = '{{ .WelcomeAnimationURL }}'`)
	assert.NotContains(t, page, `welcomeIcon.src = '{{ .GoodbyeAnimationURL }}'`)
}

// A theme swap must wait for the terminal to go idle: replacing the animation
// image mid-playback is a visible glitch, and a reload during registration would
// discard whatever the admin was typing.
func TestReaderTemplate_DefersThemeUpdatesUntilIdle(t *testing.T) {
	js := readReaderScript(t)

	// An in-place patch waits only for an animation to finish. Gating it on the
	// registration modal too would leave an unregistered terminal — where that modal
	// stays open until someone registers it — stuck on the old theme forever.
	assert.Regexp(t, `function canApplyThemeInPlace\(\) \{\s*return !animationInProgress;`, js)
	assert.Regexp(t, `function canReloadPage\(\) \{\s*return !regModalOpen && !animationInProgress;`, js)
	assert.Contains(t, js, `animationInProgress = true;`)
	// Both transitions back to idle must flush the pending state, otherwise a theme
	// change deferred during a scan would sit unapplied until the next poll.
	assert.Regexp(t, `animationInProgress = false;\s*applyPendingThemeState\(\);`, js)
	assert.Regexp(t, `rfidInput\.focus\(\);\s*applyPendingThemeState\(\);`, js)
}

// Images load asynchronously, so the revision must be recorded only after they all
// arrived. Recording it up front would mark the change as handled while a failed
// download left the terminal on stale artwork, and no later poll would retry it
// because the revision would already match.
func TestReaderTemplate_RevisionIsRecordedOnlyAfterEverythingLoaded(t *testing.T) {
	js := readReaderScript(t)
	page := readReaderPage(t)

	assert.Equal(t, 1, strings.Count(page, `themeRevision = state.revision;`))

	// The single assignment must sit after the failure guard bails out.
	guardIdx := strings.Index(js, `if (!everythingLoaded) {`)
	commitIdx := strings.Index(js, `themeRevision = state.revision;`)
	require.NotEqual(t, -1, guardIdx, "the failure guard is missing")
	assert.Less(t, guardIdx, commitIdx, "the revision must be recorded behind the failure guard")

	// It must live in applyPendingThemeState's completion callback, not in
	// applyThemeState, which returns before its images have loaded.
	applyStateIdx := strings.Index(js, `function applyThemeState(state, onDone) {`)
	callbackIdx := strings.Index(js, `applyThemeState(state, function(everythingLoaded) {`)
	require.NotEqual(t, -1, applyStateIdx)
	require.NotEqual(t, -1, callbackIdx)
	assert.Less(t, callbackIdx, commitIdx)

	// A retry must not race an apply that is still waiting on its images.
	assert.Contains(t, js, `if (!pendingThemeState || themeApplyInFlight) return;`)
	// The applied state is cleared only if a newer one has not replaced it.
	assert.Contains(t, js, `if (pendingThemeState === state) {`)
}

// A full reload is reserved for a version change: it throws away whatever the
// admin was typing into the registration form and blanks the panel while the page
// comes back, so an artwork-only change is patched in place instead.
func TestReaderTemplate_ReloadsOnlyOnAppVersionChange(t *testing.T) {
	js := readReaderScript(t)
	page := readReaderPage(t)

	assert.Equal(t, 1, strings.Count(page, `location.reload()`))
	// The single reload must sit behind both the app-version check and the
	// terminal-is-free check, in that order.
	assert.Regexp(t,
		`if \(pendingThemeState\.app_version !== appVersion\) \{\s*if \(!canReloadPage\(\)\) return;\s*location\.reload\(\);`,
		js)
}

// The kiosk sits on a LAN that may have no route to the internet, and it stays
// open for weeks. Anything the page needs must come from this binary's own
// /assets tree, or a CDN outage leaves the terminal unstyled and unusable.
func TestReaderTemplate_LoadsNoExternalResources(t *testing.T) {
	html := readReaderTemplate(t)

	// Only fetched references are checked: the inline SVG carries
	// xmlns="http://www.w3.org/2000/svg", which is a namespace name, not a URL
	// the browser ever requests.
	assert.NotContains(t, html, `src="http`)
	assert.NotContains(t, html, `href="http`)

	require.FileExists(t, websitePath("assets", "vendor", "bootstrap", "bootstrap.min.css"))
	require.FileExists(t, websitePath("assets", "vendor", "bootstrap", "bootstrap.bundle.min.js"))
}

// Terminals cache assets for weeks. Without the version query a deployed fix to
// reader.js or reader.css would never reach them.
func TestReaderTemplate_CacheBustsItsAssets(t *testing.T) {
	html := readReaderTemplate(t)

	assert.Contains(t, html, `/assets/vendor/bootstrap/bootstrap.min.css?v={{ .AppVersion }}`)
	assert.Contains(t, html, `/assets/css/reader.css?v={{ .AppVersion }}`)
	assert.Contains(t, html, `/assets/vendor/bootstrap/bootstrap.bundle.min.js?v={{ .AppVersion }}`)
	assert.Contains(t, html, `/assets/js/reader.js?v={{ .AppVersion }}`)
}

// The 800x480 Raspberry Pi panels have to show both panes at once. A Bootstrap
// grid column stacks them below 992px, which hides half the page.
func TestReaderTemplate_PanesNeverStack(t *testing.T) {
	html := readReaderTemplate(t)
	css := readReaderStyles(t)

	assert.NotContains(t, html, `class="container`)
	assert.NotContains(t, html, `class="row`)
	assert.NotContains(t, html, `class="col-`)
	assert.NotContains(t, html, `col-lg-`)

	kiosk := regexp.MustCompile(`(?s)#kiosk \{(.*?)\n\}`).FindStringSubmatch(css)
	require.Len(t, kiosk, 2, "the #kiosk rule is missing from reader.css")
	assert.Contains(t, kiosk[1], `flex-direction: row;`)
	assert.Contains(t, kiosk[1], `flex-wrap: nowrap;`)

	// Nothing may cap the page at a fixed width or height either — the panes are
	// sized from the viewport so they fit whatever panel the terminal has.
	assert.Contains(t, css, `height: 100dvh;`)
}

// Bootstrap 5 dropped jQuery. A leftover $() would throw on load and freeze the
// kiosk with no way to register the terminal.
func TestReaderScript_UsesNoJQuery(t *testing.T) {
	html := readReaderTemplate(t)
	js := readReaderScript(t)

	// Calls, not prose: the comment above the modal setup mentions jQuery on
	// purpose, to say why it is gone.
	assert.NotContains(t, js, `$(`)
	assert.NotContains(t, js, `jQuery(`)
	assert.NotContains(t, html, `jquery`)

	assert.Contains(t, js, `bootstrap.Modal.getOrCreateInstance(`)
	assert.Contains(t, js, `terminalRegModalEl.addEventListener('shown.bs.modal'`)
	assert.Contains(t, js, `terminalRegModalEl.addEventListener('hidden.bs.modal'`)
}

// Bootstrap 5 renamed every data attribute and removed .form-group and .close,
// so a stale attribute silently stops working rather than failing loudly.
func TestReaderTemplate_UsesBootstrap5Markup(t *testing.T) {
	html := readReaderTemplate(t)

	assert.NotContains(t, html, `data-dismiss=`)
	assert.NotContains(t, html, `data-backdrop=`)
	assert.NotContains(t, html, `data-keyboard=`)
	assert.NotContains(t, html, `class="close"`)
	assert.NotContains(t, html, `form-group`)

	assert.Contains(t, html, `data-bs-backdrop="static"`)
	assert.Contains(t, html, `data-bs-keyboard="false"`)

	// Dead markup no JS ever referenced: the 404 path uses #error-message and the
	// generic failure path uses alert().
	assert.NotContains(t, html, `noStudentModal`)
	assert.NotContains(t, html, `errorModal`)
}

// Only the server-rendered theme state stays inline; everything else lives in
// reader.js, where the browser can cache it and a diff can be read.
func TestReaderTemplate_InlineScriptIsOnlyTemplatedVars(t *testing.T) {
	html := readReaderTemplate(t)

	inline := regexp.MustCompile(`(?s)<script>(.*?)</script>`).FindAllStringSubmatch(html, -1)
	require.Len(t, inline, 1, "reader.html must contain exactly one inline script block")

	for _, line := range strings.Split(strings.TrimSpace(inline[0][1]), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		assert.Regexp(t, `^var \w+ +=.*;$`, line,
			"the inline block may only declare templated theme variables")
	}
}
