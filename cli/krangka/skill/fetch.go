package skill

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

const defaultRepo = "https://github.com/redhajuanda/krangka"

// repoURL returns the source repo for krangka, overridable via KRANGKA_REPO.
func repoURL() string {
	if v := os.Getenv("KRANGKA_REPO"); v != "" {
		return v
	}
	return defaultRepo
}

// resolveVersion returns the requested ref, or the latest semver-ish tag
// when version is empty.
func resolveVersion(version string) (string, error) {
	if version != "" {
		return version, nil
	}
	out, err := exec.Command("git", "ls-remote", "--tags", "--refs", repoURL()).Output()
	if err != nil {
		return "", fmt.Errorf("list remote tags: %w", err)
	}
	var tags []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		i := strings.Index(line, "refs/tags/")
		if i < 0 {
			continue
		}
		tags = append(tags, line[i+len("refs/tags/"):])
	}
	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found at %s", repoURL())
	}
	sort.Slice(tags, func(i, j int) bool { return semverLess(tags[i], tags[j]) })
	return tags[len(tags)-1], nil
}

// fetchVersion shallow-clones the repo at the given ref into a temp dir
// and returns the dir. The caller is responsible for removing it.
func fetchVersion(ref string) (string, error) {
	tmp, err := os.MkdirTemp("", "krangka-skill-*")
	if err != nil {
		return "", err
	}
	cmd := exec.Command("git", "clone", "--depth=1", "--branch", ref, repoURL(), tmp)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmp)
		return "", fmt.Errorf("clone %s@%s: %w", repoURL(), ref, err)
	}
	// Strip the .git directory — we don't want to leave it behind.
	os.RemoveAll(tmp + "/.git")
	return tmp, nil
}

// semverLess does a coarse lexical-with-numeric tag compare. Good enough
// for krangka's vMAJOR.MINOR.PATCH scheme; not full semver.
func semverLess(a, b string) bool {
	an := strings.TrimPrefix(a, "v")
	bn := strings.TrimPrefix(b, "v")
	ap := splitNums(an)
	bp := splitNums(bn)
	for i := 0; i < len(ap) && i < len(bp); i++ {
		if ap[i] != bp[i] {
			return ap[i] < bp[i]
		}
	}
	return len(ap) < len(bp)
}

func splitNums(s string) []int {
	parts := strings.Split(s, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				break
			}
			n = n*10 + int(c-'0')
		}
		out = append(out, n)
	}
	return out
}
