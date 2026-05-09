package skill

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const versionFile = ".krangka/.VERSION"

func newUpgradeCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "upgrade [version]",
		Short: "Upgrade krangka agent skills, rules, and configs to a target version",
		Long: `Sync the agent-config surface (skills, rules, MCP configs) of the
current project to a target krangka version (or to latest if omitted).

The set of paths to replace is read from the target version's
.krangka/skill-manifest.yaml. Local files listed under "exclude" are
preserved.

Use --force to skip the dirty-tree check, or --dry-run to preview.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			version := ""
			if len(args) == 1 {
				version = args[0]
			}
			return runUpgrade(version, force, dryRun)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "skip the uncommitted-changes check")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the file list without touching disk")
	return cmd
}

func runUpgrade(version string, force, dryRun bool) error {
	dst, err := os.Getwd()
	if err != nil {
		return err
	}

	current := readVersion(dst)
	target, err := resolveVersion(version)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "krangka: upgrading skills from %s to %s\n", current, target)

	src, err := fetchVersion(target)
	if err != nil {
		return err
	}
	defer os.RemoveAll(src)

	newM, err := loadManifest(src)
	if err != nil {
		return fmt.Errorf("load target manifest: %w", err)
	}
	if newM == nil {
		return fmt.Errorf("target version %s has no .krangka/skill-manifest.{yaml,json}", target)
	}
	oldM, err := localManifest()
	if err != nil {
		return fmt.Errorf("load local manifest: %w", err)
	}

	paths := unionPaths(newM, oldM)
	exclude := newM.excludeSet()

	if !force {
		dirty, err := dirtyPaths(dst, paths)
		if err == nil && len(dirty) > 0 {
			return fmt.Errorf("uncommitted changes in template-owned paths:\n  %s\ncommit, stash, or pass --force",
				strings.Join(dirty, "\n  "))
		}
	}

	if dryRun {
		fmt.Println("paths that would be replaced:")
		for _, p := range paths {
			fmt.Println("  " + p)
		}
		return nil
	}

	changes, err := replacePaths(src, dst, paths, exclude)
	if err != nil {
		return err
	}

	if _, skip := exclude[filepath.Clean(versionFile)]; !skip {
		if err := writeVersion(dst, target); err != nil {
			return fmt.Errorf("write version: %w", err)
		}
	}

	fmt.Fprintf(os.Stderr, "krangka: upgraded to %s\n", target)
	for _, c := range changes {
		fmt.Fprintln(os.Stderr, "  "+c)
	}
	return nil
}

func unionPaths(newM, oldM *Manifest) []string {
	seen := map[string]struct{}{}
	out := []string{}
	add := func(ps []string) {
		for _, p := range ps {
			p = filepath.Clean(p)
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	add(newM.allPaths())
	if oldM != nil {
		add(oldM.allPaths())
	}
	return out
}

func readVersion(root string) string {
	data, err := os.ReadFile(filepath.Join(root, versionFile))
	if err != nil {
		return "(unknown)"
	}
	return strings.TrimSpace(string(data))
}

func writeVersion(root, v string) error {
	p := filepath.Join(root, versionFile)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(v+"\n"), 0o644)
}

func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}
