package configs

import (
	"os"
	"strings"
)

var version string

// GetVersion returns the application version. If .VERSION does not exist (user
// handles versioning via git tags), returns "dev".
func GetVersion() string {
	if version == "" {
		content, err := os.ReadFile(".VERSION")
		if err != nil {
			version = "dev"
			return version
		}
		version = strings.TrimSpace(strings.TrimRight(string(content), "\n"))
	}
	return version
}
