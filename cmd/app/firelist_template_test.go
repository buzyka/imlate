package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readFireListTemplate(t *testing.T) string {
	t.Helper()
	return readWebsiteFile(t, "firelist.html")
}

func readFireListStyles(t *testing.T) string {
	t.Helper()
	return readWebsiteFile(t, "assets", "css", "firelist.css")
}

func readFireListScript(t *testing.T) string {
	t.Helper()
	return readWebsiteFile(t, "assets", "js", "firelist.js")
}

// The evacuation list is opened on a phone inside the school, on a LAN that may
// have no route to the internet, while an alarm is going off. A single
// reference to an external host turns a styled, working roll-call into a
// timeout — so no asset may come from anywhere but this binary's /assets tree.
func TestFireListTemplate_LoadsNoExternalResources(t *testing.T) {
	html := readFireListTemplate(t)

	assert.NotContains(t, html, `src="http`)
	assert.NotContains(t, html, `href="http`)

	require.FileExists(t, websitePath("assets", "css", "firelist.css"))
	require.FileExists(t, websitePath("assets", "js", "firelist.js"))
}

// A teacher's phone may have had this page cached since the previous drill.
// Without the version query a deployed fix to the stylesheet or the script
// would never reach the one device that needs it.
func TestFireListTemplate_CacheBustsItsAssets(t *testing.T) {
	html := readFireListTemplate(t)

	assert.Contains(t, html, `/assets/css/firelist.css?v={{ .AppVersion }}`)
	assert.Contains(t, html, `/assets/js/firelist.js?v={{ .AppVersion }}`)

	// Every local asset, not just the two known ones: a third file added later
	// must be cache-busted too.
	refs := regexp.MustCompile(`(?:src|href)="(/assets/[^"]*)"`).FindAllStringSubmatch(html, -1)
	require.NotEmpty(t, refs, "firelist.html references no local assets at all")
	for _, ref := range refs {
		assert.Contains(t, ref[1], `?v={{ .AppVersion }}`,
			"asset %q is not cache-busted", ref[1])
	}
}

// The page has to paint on a saturated mobile link, so its budget is three
// requests and no framework. Bootstrap or jQuery creeping back in is the exact
// regression that would break that quietly — it should fail a test, not be
// caught in review.
func TestFireListTemplate_ShipsNoFrameworks(t *testing.T) {
	html := readFireListTemplate(t)
	js := readFireListScript(t)

	assert.NotContains(t, html, `bootstrap`)
	assert.NotContains(t, html, `jquery`)
	assert.NotContains(t, html, `$(`)
	assert.NotContains(t, html, `jQuery(`)
	assert.NotContains(t, js, `$(`)
	assert.NotContains(t, js, `jQuery(`)
}

// Without a viewport meta the phone lays the page out at 980px and scales it
// down, which makes both the names and the tick targets too small to use.
func TestFireListTemplate_DeclaresMobileViewport(t *testing.T) {
	html := readFireListTemplate(t)

	assert.Contains(t, html, `<meta name="viewport"`)
	assert.Contains(t, html, `width=device-width`)
}

// Only the server-rendered identity of the list stays inline; all behaviour
// lives in firelist.js, where the browser can cache it across the reloads this
// page invites. Logic drifting back into the template would be re-downloaded
// with every refresh during an alarm.
func TestFireListTemplate_InlineScriptIsOnlyTemplatedVars(t *testing.T) {
	html := readFireListTemplate(t)

	inline := regexp.MustCompile(`(?s)<script>(.*?)</script>`).FindAllStringSubmatch(html, -1)
	require.Len(t, inline, 1, "firelist.html must contain exactly one inline script block")

	for _, line := range strings.Split(strings.TrimSpace(inline[0][1]), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		assert.Regexp(t, `^var \w+ *=.*;$`, line,
			"the inline block may only declare templated variables")
	}

	assert.Contains(t, html, `var fireListDay = '{{ .Day }}';`)
	assert.Contains(t, html, `var fireListGrade = '{{ .GradeParam }}';`)
}

// An @import is a second, serialised round trip before the page can paint, and
// a url(http…) is an external fetch the school LAN may not be able to make.
// Both defeat the three-request budget the stylesheet is written to.
func TestFireListStyles_FetchNothingExtra(t *testing.T) {
	css := readFireListStyles(t)

	assert.NotContains(t, css, `@import`)
	assert.NotContains(t, css, `url(http`)
}

// No animation, by request and by need: the phone is likely a cheap one, the
// list is repainted on every tick and every re-sort, and a transition on a
// row's background turns a 24-tap count into 24 janky frames. Enforced here so
// a "nice touch" cannot be added without the decision being reopened.
func TestFireListStyles_AnimateNothing(t *testing.T) {
	css := readFireListStyles(t)

	assert.NotContains(t, css, `transition:`)
	assert.NotContains(t, css, `animation:`)
}

// A screen reader must not announce two dozen identical unlabelled checkboxes,
// and the tick column must stay unsortable: reordering rows under the
// teacher's finger mid-count loses their place.
func TestFireListTemplate_TableIsUsableWithoutSight(t *testing.T) {
	html := readFireListTemplate(t)

	assert.Contains(t, html, `aria-label="{{ .Surname }} {{ .Name }}"`)

	sortable := regexp.MustCompile(`<th[^>]*data-sort="([^"]+)"`).FindAllStringSubmatch(html, -1)
	require.Len(t, sortable, 3, "exactly the three data columns may be sortable")
	for _, th := range sortable {
		assert.NotEqual(t, "check", th[1])
	}

	// Every sortable header must be reachable and announced as sortable.
	heads := regexp.MustCompile(`<th[^>]*data-sort=[^>]*>`).FindAllString(html, -1)
	for _, th := range heads {
		assert.Contains(t, th, `role="button"`)
		assert.Contains(t, th, `tabindex="0"`)
		assert.Contains(t, th, `aria-sort=`)
	}
}

// Legibility beats cleverness on a counted row: the name still has to be
// readable for the handover to whoever takes the list next, so a ticked row is
// filled and barred, never struck through.
func TestFireListStyles_DoNotStrikeThroughCountedRows(t *testing.T) {
	css := readFireListStyles(t)

	assert.NotContains(t, css, `line-through`)
	assert.Contains(t, css, `.fl-table tbody tr.is-checked {`)
}
