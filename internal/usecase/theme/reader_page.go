package theme

const (
	DefaultFaviconURL          = "/assets/img/favicon.ico"
	DefaultLogoBackgroundURL   = "/assets/img/default-logo.png"
	DefaultWelcomeAnimationURL = "/assets/img/welcome-images-server.gif"
	DefaultGoodbyeAnimationURL = "/assets/img/good-bye.gif"

	MinAnimationDurationMs = 1800

	DefaultWelcomeDurationMs = MinAnimationDurationMs
	DefaultGoodbyeDurationMs = MinAnimationDurationMs
)

type ReaderPageData struct {
	FaviconURL          string
	LogoBackgroundURL   string
	WelcomeAnimationURL string
	GoodbyeAnimationURL string
	WelcomeDurationMs   int
	GoodbyeDurationMs   int
}

func DefaultReaderPageData() ReaderPageData {
	return ReaderPageData{
		FaviconURL:          DefaultFaviconURL,
		LogoBackgroundURL:   DefaultLogoBackgroundURL,
		WelcomeAnimationURL: DefaultWelcomeAnimationURL,
		GoodbyeAnimationURL: DefaultGoodbyeAnimationURL,
		WelcomeDurationMs:   DefaultWelcomeDurationMs,
		GoodbyeDurationMs:   DefaultGoodbyeDurationMs,
	}
}
