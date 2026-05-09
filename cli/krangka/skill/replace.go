package skill

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// replacePaths replaces the given paths in dst with their counterparts
// from src. Missing src paths are deleted from dst. Excluded paths are
// preserved. Symlinks are recreated, except on Windows where they are
// replaced with copies.
func replacePaths(src, dst string, paths []string, exclude map[string]struct{}) (changes []string, err error) {
	for _, rel := range paths {
		rel = filepath.Clean(rel)
		srcPath := filepath.Join(src, rel)
		dstPath := filepath.Join(dst, rel)

		if _, skip := exclude[rel]; skip {
			continue
		}

		srcInfo, srcErr := os.Lstat(srcPath)
		dstExists := exists(dstPath)

		switch {
		case errors.Is(srcErr, fs.ErrNotExist):
			if dstExists {
				if err := os.RemoveAll(dstPath); err != nil {
					return changes, fmt.Errorf("remove %s: %w", dstPath, err)
				}
				changes = append(changes, "removed "+rel)
			}
		case srcErr != nil:
			return changes, fmt.Errorf("stat %s: %w", srcPath, srcErr)
		default:
			if dstExists {
				if err := os.RemoveAll(dstPath); err != nil {
					return changes, fmt.Errorf("clean %s: %w", dstPath, err)
				}
			}
			if err := copyAny(srcPath, dstPath, srcInfo, exclude, rel); err != nil {
				return changes, err
			}
			if dstExists {
				changes = append(changes, "updated "+rel)
			} else {
				changes = append(changes, "added   "+rel)
			}
		}
	}
	return changes, nil
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

func copyAny(src, dst string, info os.FileInfo, exclude map[string]struct{}, baseRel string) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return copySymlink(src, dst)
	}
	if info.IsDir() {
		return copyDir(src, dst, exclude, baseRel)
	}
	return copyFile(src, dst, info.Mode())
}

// copySymlink resolves the symlink target and copies it as a real file
// or directory into dst. Template symlinks (e.g. .krangka/README.md ->
// ../README.md) must be materialized; recreating the symlink in the
// destination project would resolve against the user's repo root and
// pick up unrelated content.
func copySymlink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return fmt.Errorf("readlink %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	resolved := target
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(src), target)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("resolve symlink %s -> %s: %w", src, target, err)
	}
	if info.IsDir() {
		return copyDir(resolved, dst, nil, "")
	}
	return copyFile(resolved, dst, info.Mode())
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string, exclude map[string]struct{}, baseRel string) error {
	return filepath.Walk(src, func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		// Skip excluded paths within this dir.
		if baseRel != "" && exclude != nil {
			full := filepath.ToSlash(filepath.Join(baseRel, rel))
			if _, skip := exclude[filepath.Clean(full)]; skip {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		target := filepath.Join(dst, rel)
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return copySymlink(p, target)
		case info.IsDir():
			return os.MkdirAll(target, info.Mode())
		default:
			return copyFile(p, target, info.Mode())
		}
	})
}

// dirtyPaths returns the subset of paths that have uncommitted changes
// in dst's git working tree.
func dirtyPaths(dst string, paths []string) ([]string, error) {
	out, err := runGit(dst, "status", "--porcelain", "-z")
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	var dirty []string
	for _, entry := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if len(entry) < 4 {
			continue
		}
		path := entry[3:]
		for _, p := range paths {
			p = filepath.Clean(p)
			if path == p || strings.HasPrefix(path, p+"/") {
				dirty = append(dirty, path)
				break
			}
		}
	}
	return dirty, nil
}
