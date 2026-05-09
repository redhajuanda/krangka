package skill

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed skill-manifest.yaml
var embeddedManifest []byte

type Manifest struct {
	Version int      `yaml:"version" json:"version"`
	Paths   Paths    `yaml:"paths" json:"paths"`
	Exclude []string `yaml:"exclude" json:"exclude"`
}

type Paths struct {
	Dirs  []string `yaml:"dirs" json:"dirs"`
	Files []string `yaml:"files" json:"files"`
}

const (
	manifestYAML = "cli/krangka/skill/skill-manifest.yaml"
	manifestJSON = "cli/krangka/skill/skill-manifest.json"
)

// localManifest returns the manifest baked into this CLI binary. It
// represents "what the currently installed krangka CLI considers
// template-owned" and is used to detect orphan paths when upgrading.
func localManifest() (*Manifest, error) {
	m := &Manifest{}
	if err := yaml.Unmarshal(embeddedManifest, m); err != nil {
		return nil, fmt.Errorf("parse embedded manifest: %w", err)
	}
	if m.Version == 0 {
		return nil, fmt.Errorf("embedded manifest: missing version")
	}
	return m, nil
}

// loadManifest looks for the manifest under root, preferring YAML.
// Returns (nil, nil) if no manifest is present.
func loadManifest(root string) (*Manifest, error) {
	for _, name := range []string{manifestYAML, manifestJSON} {
		p := filepath.Join(root, name)
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", p, err)
		}
		m := &Manifest{}
		if strings.HasSuffix(name, ".json") {
			if err := json.Unmarshal(data, m); err != nil {
				return nil, fmt.Errorf("parse %s: %w", p, err)
			}
		} else {
			if err := yaml.Unmarshal(data, m); err != nil {
				return nil, fmt.Errorf("parse %s: %w", p, err)
			}
		}
		if m.Version == 0 {
			return nil, fmt.Errorf("%s: missing or zero `version` field", p)
		}
		return m, nil
	}
	return nil, nil
}

// allPaths returns dirs and files concatenated.
func (m *Manifest) allPaths() []string {
	out := make([]string, 0, len(m.Paths.Dirs)+len(m.Paths.Files))
	out = append(out, m.Paths.Dirs...)
	out = append(out, m.Paths.Files...)
	return out
}

// excludeSet returns excluded paths as a lookup map.
func (m *Manifest) excludeSet() map[string]struct{} {
	s := make(map[string]struct{}, len(m.Exclude))
	for _, p := range m.Exclude {
		s[filepath.Clean(p)] = struct{}{}
	}
	return s
}
