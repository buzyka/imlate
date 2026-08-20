package theme

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/buzyka/imlate/internal/version"
)

const (
	DefaultFaviconURL          = "/assets/img/favicon.ico"
	DefaultLogoBackgroundURL   = "/assets/img/default-logo.png"
	DefaultWelcomeAnimationURL = "/assets/img/welcome-images-server.gif"
	DefaultGoodbyeAnimationURL = "/assets/img/good-bye.gif"

	MinAnimationDurationMs = 1800

	DefaultWelcomeDurationMs = MinAnimationDurationMs
	DefaultGoodbyeDurationMs = MinAnimationDurationMs

	// DefaultReaderThemePollSeconds is how often the tracking page asks the
	// server whether its theme changed.
	DefaultReaderThemePollSeconds = 300
	// minReaderThemePollSeconds guards against a misconfigured
	// READER_THEME_POLL_SECONDS turning every terminal into a request generator.
	minReaderThemePollSeconds = 10
	// readerRevisionLength is how much of the fingerprint hash is kept. The value
	// only has to distinguish one theme state from another, not resist attack.
	readerRevisionLength = 16
)

// ReaderPageData is both the template data for the tracking page and the JSON
// payload of the public theme-state endpoint, so the page and the endpoint can
// never disagree about what the current theme is.
type ReaderPageData struct {
	FaviconURL          string `json:"favicon_url"`
	LogoBackgroundURL   string `json:"logo_background_url"`
	WelcomeAnimationURL string `json:"welcome_animation_url"`
	GoodbyeAnimationURL string `json:"goodbye_animation_url"`
	WelcomeDurationMs   int    `json:"welcome_duration_ms"`
	GoodbyeDurationMs   int    `json:"goodbye_duration_ms"`

	Revision string `json:"revision"`
	AppVersion string `json:"app_version"`
	// PollSeconds is how often the page should poll for changes.
	PollSeconds int `json:"poll_seconds"`
}

func DefaultReaderPageData() ReaderPageData {
	data := ReaderPageData{
		FaviconURL:          DefaultFaviconURL,
		LogoBackgroundURL:   DefaultLogoBackgroundURL,
		WelcomeAnimationURL: DefaultWelcomeAnimationURL,
		GoodbyeAnimationURL: DefaultGoodbyeAnimationURL,
		WelcomeDurationMs:   DefaultWelcomeDurationMs,
		GoodbyeDurationMs:   DefaultGoodbyeDurationMs,
	}
	return finalizeReaderPageData(data, DefaultReaderThemePollSeconds)
}

func finalizeReaderPageData(data ReaderPageData, pollSeconds int) ReaderPageData {
	data.Revision = readerRevision(data)
	data.AppVersion = version.Version
	data.PollSeconds = normalizePollSeconds(pollSeconds)
	return data
}

func normalizePollSeconds(pollSeconds int) int {
	if pollSeconds <= 0 {
		return DefaultReaderThemePollSeconds
	}
	if pollSeconds < minReaderThemePollSeconds {
		return minReaderThemePollSeconds
	}
	return pollSeconds
}

func readerRevision(data ReaderPageData) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\n%s\n%s\n%s\n%d\n%d",
		data.FaviconURL,
		data.LogoBackgroundURL,
		data.WelcomeAnimationURL,
		data.GoodbyeAnimationURL,
		data.WelcomeDurationMs,
		data.GoodbyeDurationMs,
	)
	return hex.EncodeToString(h.Sum(nil))[:readerRevisionLength]
}
