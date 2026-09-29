package version

var (
	// Version is the current application semantic version.
	// Can be overridden at build time via -ldflags="-X 'tgbot/internal/version.Version=1.0.0'".
	Version = "1.0.0"

	// Commit is the git commit hash, set at build time.
	Commit = "dev"

	// BuildDate is the date/time when binary was built, set at build time.
	BuildDate = "unknown"
)

// String returns formatted version.
func String() string {
	return Version
}

// FullInfo returns a multi-line or formatted version info string.
func FullInfo() string {
	return Version + " (" + Commit + ", " + BuildDate + ")"
}
