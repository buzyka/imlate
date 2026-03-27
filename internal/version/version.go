package version

// Version is the application version. Overridden at build time via:
//
//	go build -ldflags "-X github.com/buzyka/imlate/internal/version.Version=2.0.0"
var Version = "2.0.x-dev"
