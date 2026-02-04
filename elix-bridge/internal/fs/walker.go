package fs

import (
	"os"
	"path/filepath"
)

// Walker provides optimized file system traversal
type Walker struct {
	BaseDir string
}

func NewWalker(baseDir string) *Walker {
	return &Walker{
		BaseDir: baseDir,
	}
}

// Configurable ignore list
var ignoredDirs = map[string]bool{
	".git":         true,
	".svn":         true,
	".hg":          true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"bin":          true,
	"obj":          true,
	"target":       true,
	".idea":        true,
	".vscode":      true,
	"venv":         true,
	".venv":        true,
	"env":          true,
	".env":         true,
	"__pycache__":  true,
}

// FileEntry represents a file or directory in the list
type FileEntry struct {
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size,omitempty"`
}

// ListFiles traverses the directory and returns a list of files
func (w *Walker) ListFiles(relPath string, recursive bool) ([]FileEntry, error) {
	rootPath := filepath.Join(w.BaseDir, relPath)
	var entries []FileEntry

	// If recursive, we use WalkDir (more memory efficient than Walk)
	if recursive {
		err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				// Skip unreadable files/dirs but continue walking
				return nil
			}

			// Get relative path from request root (or BaseDir)
			// For the API response, we want paths relative to the requested root
			// relToRequest, _ := filepath.Rel(rootPath, path)

			// Actually, for @ inputs, simpler is better: return path relative to Project Root (BaseDir)
			relToProject, err := filepath.Rel(w.BaseDir, path)
			if err != nil {
				return nil
			}

			// Normalize to forward slashes for consistency across OS
			relToProject = filepath.ToSlash(relToProject)

			if d.IsDir() {
				if ignoredDirs[d.Name()] {
					return filepath.SkipDir
				}
				// Don't include the root itself in the list
				if path == rootPath {
					return nil
				}
			}

			entries = append(entries, FileEntry{
				Path:  relToProject,
				IsDir: d.IsDir(),
				// Getting size requires Info(), which is an extra stat call.
				// For WalkDir, DirEntry usually has Info cached on Linux/Windows?
				// Actually DirEntry.Info() might cause a stat.
				// For high perf fuzzy search, we might not need size immediately.
				// Let's optimize speed for now and skip Size unless it's cheap.
			})

			return nil
		})
		return entries, err
	}

	// Non-recursive (readdir)
	dirEntries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, err
	}

	for _, d := range dirEntries {
		relToProject, _ := filepath.Rel(w.BaseDir, filepath.Join(rootPath, d.Name()))
		relToProject = filepath.ToSlash(relToProject)

		entries = append(entries, FileEntry{
			Path:  relToProject,
			IsDir: d.IsDir(),
		})
	}

	return entries, nil
}
