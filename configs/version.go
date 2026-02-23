package configs

import (
	"log"
	"os"
	"strings"
)

var version string

func GetVersion() string {
	if version == "" {
		content, err := os.ReadFile(".VERSION")
		if err != nil {
			log.Fatal(err)
		}
		version = strings.TrimSpace(strings.TrimRight(string(content), "\n"))
	}
	return version
}
