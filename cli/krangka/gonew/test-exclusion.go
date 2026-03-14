package gonew

import (
	"fmt"
)

// TestDirectoryExclusion tests the directory exclusion functionality
func TestDirectoryExclusion() {
	config := loadExclusionConfig()

	testPaths := []string{
		"cli",
		"internal/adapter/outbound/postgres",
		"internal/core/service/note",
		"internal/core/domain/note.go",
		"main.go",
		"go.mod",
	}

	fmt.Println("Testing directory and file exclusions:")
	for _, testPath := range testPaths {
		isDir := !contains(testPath, ".") || testPath == "go.mod"
		excluded := shouldExclude(testPath, isDir, config)
		fmt.Printf("  %s (isDir: %t) -> excluded: %t\n", testPath, isDir, excluded)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && s[len(substr)] == '/' || s[len(substr)] == '.')
}
