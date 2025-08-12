package scanner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/grantbirki/gh-pin/internal/processor"
	"github.com/regclient/regclient"
)

// isStandardDockerfile checks if a filename matches standard Dockerfile patterns
// during recursive scanning (more restrictive)
func isStandardDockerfile(filename string) bool {
	lower := strings.ToLower(filename)

	// Exact matches for standard names
	if lower == "dockerfile" {
		return true
	}

	// Standard .dockerfile extension
	if strings.HasSuffix(lower, ".dockerfile") {
		return true
	}

	// Standard Dockerfile with dot suffixes (Dockerfile.dev, Dockerfile.prod, etc.)
	if strings.HasPrefix(lower, "dockerfile.") {
		return true
	}

	return false
}

// ScanPath walks a directory and processes all supported files
func ScanPath(rc *regclient.RegClient, root string, config processor.ProcessorConfig, recursive bool) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}

		basename := filepath.Base(path)
		lower := strings.ToLower(basename)
		ext := strings.ToLower(filepath.Ext(path))

		switch {
		case isStandardDockerfile(basename):
			fmt.Printf("Processing Dockerfile: %s\n", path)
			return processor.ProcessDockerfile(rc, path, config)
		case lower == "docker-compose.yml",
			lower == "docker-compose.yaml",
			(ext == ".yml" || ext == ".yaml"):
			// Try parsing as compose; skip if not valid
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var cf processor.ComposeFile
			if err := yaml.Unmarshal(data, &cf); err != nil {
				return nil // skip non-compose YAMLs
			}
			if len(cf.Services) > 0 {
				fmt.Printf("Processing Compose: %s\n", path)
				return processor.ProcessCompose(rc, path, config)
			}
		}
		return nil
	})
}

// ProcessSingleFile processes a single file based on its type
// More permissive than directory scanning since user explicitly specified the file
func ProcessSingleFile(rc *regclient.RegClient, target string, config processor.ProcessorConfig) error {
	basename := filepath.Base(target)
	lower := strings.ToLower(basename)
	ext := strings.ToLower(filepath.Ext(target))

	switch {
	// More permissive Dockerfile detection for explicit file paths
	case lower == "dockerfile" ||
		strings.HasSuffix(lower, ".dockerfile") ||
		strings.HasPrefix(lower, "dockerfile"):
		return processor.ProcessDockerfile(rc, target, config)
	case ext == ".yml" || ext == ".yaml":
		return processor.ProcessCompose(rc, target, config)
	default:
		fmt.Printf("Skipping unsupported file: %s\n", target)
		return nil
	}
}
