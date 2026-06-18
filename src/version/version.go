package version

import "strings"

var (
	// GitSummary is populated at compile time by ldflags (see Makefile)
	GitSummary string
)

// GetVersion returns the fully described git version of this bot.
func GetVersion() string {
	if GitSummary == "" {
		return "N/A"
	}
	return strings.ReplaceAll(GitSummary, "dirty", "edge")
}
