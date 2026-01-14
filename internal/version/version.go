package version

// Version is set via ldflags during build
// Example: go build -ldflags "-X github.com/mtzanidakis/budgeting/internal/version.Version=abc123"
var Version = "dev"

// Get returns the current version
func Get() string {
	if Version == "" {
		return "dev"
	}
	return Version
}
