package gonew

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/redhajuanda/krangka/cli/krangka/utils/stringx"
	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

//go:embed exclude-config.json
var excludeConfigFS embed.FS

// PartialDir specifies which files inside a directory should be copied; everything else is skipped.
// Matched files are always overwritten (never treated as conflicts when target is ".").
type PartialDir struct {
	Include []string `json:"include"` // file patterns (matched against path relative to the dir)
}

// ExclusionConfig holds configuration for what to exclude
type ExclusionConfig struct {
	Files              []string              `json:"files"`                 // File patterns to exclude (glob patterns)
	Directories        []string              `json:"directories"`           // Directory patterns to exclude
	CodeBlocks         []string              `json:"code_blocks"`           // Code block markers to exclude
	CurrentDirSkipDirs []string              `json:"current_dir_skip_dirs"` // Dirs silently skipped (no conflict check, no copy) when target is "."
	PartialDirs        map[string]PartialDir `json:"partial_dirs"`          // Dirs where only listed files are copied (always overwritten)
}

var (
	gonewCmd = &cobra.Command{
		Use:     "new",
		Example: "krangka new <package name> <project name> \nkrangka new github.com/redhajuanda/krangka myproject\nkrangka new github.com/redhajuanda/krangka .\n\nNote: <project name> will replace 'krangka' in deployment files like development.yaml\nUse '.' as <project name> to populate the boilerplate in the current directory",
		Short:   "Generate new project",
		Args:    cobra.MinimumNArgs(1),
		RunE:    initNew,
	}
	srcMod string
)

func Commands() []*cobra.Command {
	gonewCmd.PersistentFlags().StringVarP(&srcMod, "source mod", "m", "github.com/redhajuanda/krangka", "based boilerplate code")
	return []*cobra.Command{gonewCmd}
}

// inPartialDir checks whether rel is inside a partial_dirs entry.
// Returns (inPartial, include):
//   - inPartial=false → path is not governed by partial_dirs at all
//   - inPartial=true, include=true  → copy/overwrite this file (or allow dir traversal)
//   - inPartial=true, include=false → skip this file
func inPartialDir(rel string, isDir bool, config *ExclusionConfig) (inPartial bool, include bool) {
	parts := strings.SplitN(rel, string(filepath.Separator), 2)
	if len(parts) < 2 {
		// Top-level entry (the partial dir root itself) — not governed at file level
		return false, false
	}
	pd, ok := config.PartialDirs[parts[0]]
	if !ok {
		return false, false
	}
	if isDir {
		return true, false // subdirs inside partial dirs are always skipped
	}
	relWithin := parts[1]
	for _, pattern := range pd.Include {
		if matched, _ := filepath.Match(pattern, relWithin); matched {
			return true, true
		}
		if matched, _ := filepath.Match(pattern, filepath.Base(relWithin)); matched {
			return true, true
		}
	}
	return true, false
}

// shouldExclude checks if a file or directory should be excluded based on patterns
func shouldExclude(path string, isDir bool, config *ExclusionConfig) bool {
	// Check directory exclusions
	if isDir {
		for _, pattern := range config.Directories {
			// Check both the full path and the base name
			if matched, _ := filepath.Match(pattern, path); matched {
				return true
			}
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				return true
			}
		}
		return false
	}

	// Check file exclusions
	for _, pattern := range config.Files {
		// Check both the full path and the base name
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}

	return false
}

// replaceProjectName replaces "krangka" with the project name in specific files
func replaceProjectName(data []byte, filename string, projectName string) []byte {
	// Files that should have "krangka" replaced with project name
	targetFiles := []string{
		"configs/files/example.yaml",
		"development_main.yaml",
		"development_kafka.yaml",
		"docker-compose.yml",
		"docker-compose.yaml",
		"deployment.yaml",
		"deployment.yml",
		"Makefile",
	}

	// Check if this file should have project name replacement
	baseName := filepath.Base(filename)
	shouldReplace := false
	for _, target := range targetFiles {
		if baseName == target || filename == target {
			shouldReplace = true
			break
		}
	}

	// Also check if the file is in a deployment directory
	if !shouldReplace && (strings.Contains(filename, "deployment/") || strings.Contains(filename, "deploy/")) {
		for _, target := range targetFiles {
			if baseName == target {
				shouldReplace = true
				break
			}
		}
	}

	if !shouldReplace {
		return data
	}

	// Special handling for Makefile
	if baseName == "Makefile" {
		// Only replace BINARY=krangka line specifically
		result := string(data)
		result = strings.ReplaceAll(result, "BINARY=krangka", "BINARY="+projectName)
		return []byte(result)
	}

	// Replace all occurrences of "krangka" with the project name
	// Handle various cases: krangka, KRANGKA, Krangka
	result := bytes.ReplaceAll(data, []byte("krangka"), []byte(projectName))
	result = bytes.ReplaceAll(result, []byte("KRANGKA"), []byte(strings.ToUpper(projectName)))

	// Capitalize first letter for title case
	titleProjectName := projectName
	if len(projectName) > 0 {
		titleProjectName = strings.ToUpper(string(projectName[0])) + projectName[1:]
	}
	result = bytes.ReplaceAll(result, []byte("Krangka"), []byte(titleProjectName))

	return result
}

// moveToKrangkaFolder moves README.md and docs directory to .krangka folder and creates new README.md
func moveToKrangkaFolder(projectDir, projectName string) error {
	krangkaDir := filepath.Join(projectDir, ".krangka")

	// Create .krangka directory
	if err := os.MkdirAll(krangkaDir, 0755); err != nil {
		return fmt.Errorf("failed to create .krangka directory: %v", err)
	}

	// Move README.md to .krangka/README.md
	readmeSrc := filepath.Join(projectDir, "README.md")
	readmeDst := filepath.Join(krangkaDir, "README.md")
	if _, err := os.Stat(readmeSrc); err == nil {
		if err := os.Rename(readmeSrc, readmeDst); err != nil {
			return fmt.Errorf("failed to move README.md: %v", err)
		}
	}

	// Move docs directory to .krangka/docs
	docsSrc := filepath.Join(projectDir, "docs")
	docsDst := filepath.Join(krangkaDir, "docs")
	if _, err := os.Stat(docsSrc); err == nil {
		if err := os.Rename(docsSrc, docsDst); err != nil {
			return fmt.Errorf("failed to move docs directory: %v", err)
		}
	}

	// Create new simple README.md with just the project name
	newReadmeContent := fmt.Sprintf("# %s\n", projectName)
	newReadmePath := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(newReadmePath, []byte(newReadmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create new README.md: %v", err)
	}

	return nil
}

// extractBasePath extracts the base path from module name
// e.g., github.com/redhajuanda/krangka -> /platform/krangka
func extractBasePath(moduleName string) string {
	// Remove .git suffix if present
	cleanModule := strings.TrimSuffix(moduleName, ".git")

	// Split by slash and take the last 2 parts
	parts := strings.Split(cleanModule, "/")
	if len(parts) >= 2 {
		return "/" + strings.Join(parts[len(parts)-2:], "/")
	}

	// Fallback: if less than 2 parts, return the whole thing with leading slash
	return "/" + cleanModule
}

// updateMainGoComments updates Swagger/OpenAPI comments in main.go
func updateMainGoComments(data []byte, projectName, dstMod string) []byte {
	result := data

	// Update @title comment
	titlePattern := regexp.MustCompile(`(@title\s+)Krangka(\s+Service\s+API)`)
	result = titlePattern.ReplaceAll(result, []byte(fmt.Sprintf("${1}%s${2}", projectName)))

	// Update @description comment
	descPattern := regexp.MustCompile(`(@description\s+This\s+is\s+a\s+documentation\s+for\s+)Krangka(\s+Service\s+RESTful\s+APIs\.)`)
	result = descPattern.ReplaceAll(result, []byte(fmt.Sprintf("${1}%s${2}", projectName)))

	// Update @BasePath comment
	basePath := extractBasePath(dstMod)
	basePathPattern := regexp.MustCompile(`(@BasePath\s+)/platform/krangka`)
	result = basePathPattern.ReplaceAll(result, []byte(fmt.Sprintf("${1}%s", basePath)))

	return result
}

// removeCodeBlocks removes code blocks marked with exclusion comments
func removeCodeBlocks(data []byte, config *ExclusionConfig) []byte {
	if len(config.CodeBlocks) == 0 {
		return data
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	var result []string
	inExcludedBlock := false
	excludeStartPattern := regexp.MustCompile(`^\s*//\s*exclude-start\s*$`)
	excludeEndPattern := regexp.MustCompile(`^\s*//\s*exclude-end\s*$`)

	for scanner.Scan() {
		line := scanner.Text()

		if excludeStartPattern.MatchString(line) {
			inExcludedBlock = true
			continue
		}

		if excludeEndPattern.MatchString(line) {
			inExcludedBlock = false
			continue
		}

		if !inExcludedBlock {
			result = append(result, line)
		}
	}

	return []byte(strings.Join(result, "\n"))
}

// loadExclusionConfig loads the embedded exclusion configuration
func loadExclusionConfig() *ExclusionConfig {
	// Default configuration as fallback
	defaultConfig := &ExclusionConfig{
		Directories: []string{
			"cli",
			".git",
			".vscode",
			".idea",
			"node_modules",
			"vendor",
		},
		Files: []string{
			"*.log",
			"*.tmp",
			".env*",
			".gitattributes",
			"*.swp",
			"*.swo",
			"*~",
			".DS_Store",
			"Thumbs.db",
			"internal/core/domain/note.go",
		},
		CodeBlocks: []string{
			"exclude-start",
			"exclude-end",
		},
		CurrentDirSkipDirs: []string{
			".cursor",
			".agent",
			".claude",
		},
		PartialDirs: map[string]PartialDir{
			"openspec": {Include: []string{"config.yaml"}},
		},
	}

	// Try to load from embedded config file
	if data, err := excludeConfigFS.ReadFile("exclude-config.json"); err == nil {
		if err := json.Unmarshal(data, defaultConfig); err != nil {
			log.Printf("Warning: could not parse embedded exclusion config: %v", err)
		}
	} else {
		log.Printf("Warning: could not read embedded exclusion config: %v", err)
	}

	// Try to load from local config file if it exists (for customization)
	configPath := "exclude-config.json"
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, defaultConfig); err != nil {
			log.Printf("Warning: could not parse local exclusion config file %s: %v", configPath, err)
		}
	}

	return defaultConfig
}

func initNew(cmd *cobra.Command, args []string) error {
	srcModVers := srcMod
	if !strings.Contains(srcModVers, "@") {
		srcModVers += "@latest"
	}

	srcMod, _, _ = strings.Cut(srcModVers, "@")
	if err := module.CheckPath(srcMod); err != nil {
		log.Fatalf("invalid source module name: %v", err)
	}

	dstMod := srcMod
	if len(args) >= 1 {
		dstMod = args[0]
		if err := module.CheckPath(dstMod); err != nil {
			log.Fatalf("invalid destination module name: %v", err)
		}
	}

	var dir string
	if len(args) == 2 {
		dir = args[1]
	} else {
		dir = "." + string(filepath.Separator) + path.Base(dstMod)
	}

	// Extract project name from directory path
	var projectName string
	if dir == "." {
		// When using current directory, get the directory name from the current working directory
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("failed to get current working directory: %v", err)
		}
		projectName = filepath.Base(cwd)
	} else {
		projectName = filepath.Base(dir)
	}

	// Dir must not exist or must be an empty directory.
	// Allow "." to populate in current directory even if it has files
	de, err := os.ReadDir(dir)
	if dir != "." && err == nil && len(de) > 0 {
		log.Fatalf("target directory %s exists and is non-empty", dir)
	}
	needMkdir := err != nil

	var stdout, stderr bytes.Buffer
	gocmd := exec.Command("go", "mod", "download", "-json", srcModVers)
	gocmd.Env = append(os.Environ(), "GOPROXY=direct")
	gocmd.Stdout = &stdout
	gocmd.Stderr = &stderr
	if err := gocmd.Run(); err != nil {
		log.Fatalf("go mod download -json %s: %v\n%s%s", srcModVers, err, stderr.Bytes(), stdout.Bytes())
	}

	var info struct {
		Dir string
	}
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		log.Fatalf("go mod download -json %s: invalid JSON output: %v\n%s%s", srcMod, err, stderr.Bytes(), stdout.Bytes())
	}

	if needMkdir {
		if err := os.MkdirAll(dir, 0777); err != nil {
			log.Fatal(err)
		}
	}

	// Load exclusion configuration
	exclusionConfig := loadExclusionConfig()

	// Build lookup set from configurable current_dir_skip_dirs.
	currentDirSkipDirs := make(map[string]bool, len(exclusionConfig.CurrentDirSkipDirs))
	for _, d := range exclusionConfig.CurrentDirSkipDirs {
		currentDirSkipDirs[d] = true
	}

	// When using current directory, check for conflicting files before writing anything.
	if dir == "." {
		var conflicts []string
		filepath.WalkDir(info.Dir, func(src string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, err := filepath.Rel(info.Dir, src)
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if currentDirSkipDirs[rel] {
					return filepath.SkipDir
				}
				return nil
			}
			if shouldExclude(rel, false, exclusionConfig) {
				return nil
			}
			// Also skip files whose top-level directory is in the skip set.
			if currentDirSkipDirs[strings.SplitN(rel, string(filepath.Separator), 2)[0]] {
				return nil
			}
			// Partial-dir files are always overwritten — not a conflict.
			if inPD, include := inPartialDir(rel, false, exclusionConfig); inPD {
				if !include {
					return nil // non-included file in partial dir → skip entirely
				}
				return nil // included file → will be overwritten, not a conflict
			}
			if _, statErr := os.Stat(filepath.Join(dir, rel)); statErr == nil {
				conflicts = append(conflicts, rel)
			}
			return nil
		})
		if len(conflicts) > 0 {
			return fmt.Errorf("cannot initialize in current directory: conflicting files exist:\n  %s", strings.Join(conflicts, "\n  "))
		}
	}

	// Copy from module cache into new directory, making edits as needed.
	filepath.WalkDir(info.Dir, func(src string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		rel, err := filepath.Rel(info.Dir, src)
		if err != nil {
			log.Fatal(err)
		}

		// When initializing in current directory, skip tool-config dirs entirely.
		if dir == "." && d.IsDir() && currentDirSkipDirs[rel] {
			return filepath.SkipDir
		}

		// Handle partial dirs: only copy files matching the include list.
		if inPD, include := inPartialDir(rel, d.IsDir(), exclusionConfig); inPD {
			if d.IsDir() {
				return filepath.SkipDir // skip subdirectories inside partial dirs
			}
			if !include {
				return nil // skip non-included files in partial dirs
			}
			// included files fall through to normal copy logic below
		}

		// Check if this file/directory should be excluded
		if shouldExclude(rel, d.IsDir(), exclusionConfig) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		dst := filepath.Join(dir, rel)
		if d.IsDir() {
			if err := os.MkdirAll(dst, 0777); err != nil {
				log.Fatal(err)
			}
			return nil
		}

		data, err := os.ReadFile(src)
		if err != nil {
			log.Fatal(err)
		}

		// Remove excluded code blocks
		data = removeCodeBlocks(data, exclusionConfig)

		// Replace project name in specific files
		data = replaceProjectName(data, rel, projectName)

		isRoot := !strings.Contains(rel, string(filepath.Separator))
		if strings.HasSuffix(rel, ".go") {
			data = FixGo(data, rel, srcMod, dstMod, isRoot)
			// Update main.go with project-specific Swagger comments
			if rel == "main.go" {
				data = updateMainGoComments(data, projectName, dstMod)
			}
		}
		if rel == "go.mod" {
			data = fixGoMod(data, srcMod, dstMod)
		}

		if err := os.WriteFile(dst, data, 0666); err != nil {
			log.Fatal(err)
		}
		return nil
	})

	// Post-processing: move README.md and docs to .krangka folder
	if err := moveToKrangkaFolder(dir, projectName); err != nil {
		log.Printf("Warning: failed to organize krangka files: %v", err)
	}

	log.Printf("initialized %s in %s\n", dstMod, dir)
	return nil
}

// FixGo rewrites the Go source in data to replace srcMod with dstMod.
// isRoot indicates whether the file is in the root directory of the module,
// in which case we also update the package name.
func FixGo(data []byte, file string, srcMod, dstMod string, isRoot bool) []byte {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, data, parser.ImportsOnly)
	if err != nil {
		log.Fatalf("parsing source module:\n%s", err)
	}

	buf := stringx.NewBuffer(data)
	at := func(p token.Pos) int {
		return fset.File(p).Offset(p)
	}

	srcName := path.Base(srcMod)
	dstName := path.Base(dstMod)
	if isRoot {
		if name := f.Name.Name; name == srcName || name == srcName+"_test" {
			dname := dstName + strings.TrimPrefix(name, srcName)
			if !token.IsIdentifier(dname) {
				log.Fatalf("%s: cannot rename package %s to package %s: invalid package name", file, name, dname)
			}
			buf.Replace(at(f.Name.Pos()), at(f.Name.End()), dname)
		}
	}

	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		if path == srcMod {
			if srcName != dstName && spec.Name == nil {
				// Add package rename because source code uses original name.
				// The renaming looks strange, but template authors are unlikely to
				// create a template where the root package is imported by packages
				// in subdirectories, and the renaming at least keeps the code working.
				// A more sophisticated approach would be to rename the uses of
				// the package identifier in the file too, but then you have to worry about
				// name collisions, and given how unlikely this is, it doesn't seem worth
				// trying to clean up the file that way.
				buf.Insert(at(spec.Path.Pos()), srcName+" ")
			}
			// Change import path to dstMod
			buf.Replace(at(spec.Path.Pos()), at(spec.Path.End()), strconv.Quote(dstMod))
		}
		if strings.HasPrefix(path, srcMod+"/") {
			// Change import path to begin with dstMod
			buf.Replace(at(spec.Path.Pos()), at(spec.Path.End()), strconv.Quote(strings.Replace(path, srcMod, dstMod, 1)))
		}
	}

	// Also do a comprehensive search and replace for any remaining occurrences
	// of the module path in the entire file content
	result := buf.Bytes()
	result = bytes.ReplaceAll(result, []byte(srcMod), []byte(dstMod))

	// Handle submodule imports (e.g., srcMod/subpackage -> dstMod/subpackage)
	srcModSlash := srcMod + "/"
	dstModSlash := dstMod + "/"
	result = bytes.ReplaceAll(result, []byte(srcModSlash), []byte(dstModSlash))

	return result
}

// fixGoMod rewrites the go.mod content in data to replace srcMod with dstMod
// in the module path.
func fixGoMod(data []byte, srcMod, dstMod string) []byte {
	f, err := modfile.ParseLax("go.mod", data, nil)
	if err != nil {
		log.Fatalf("parsing source module:\n%s", err)
	}
	f.AddModuleStmt(dstMod)
	new, err := f.Format()
	if err != nil {
		return data
	}
	return new
}
