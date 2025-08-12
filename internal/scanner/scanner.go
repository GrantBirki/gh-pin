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

		lower := strings.ToLower(filepath.Base(path))
		ext := strings.ToLower(filepath.Ext(path))

		switch {
		case lower == "dockerfile" || strings.HasSuffix(lower, ".dockerfile"):
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
func ProcessSingleFile(rc *regclient.RegClient, target string, config processor.ProcessorConfig) error {
	lower := strings.ToLower(filepath.Base(target))
	ext := strings.ToLower(filepath.Ext(target))

	switch {
	case lower == "dockerfile" || strings.HasSuffix(lower, ".dockerfile"):
		return processor.ProcessDockerfile(rc, target, config)
	case ext == ".yml" || ext == ".yaml":
		return processor.ProcessCompose(rc, target, config)
	default:
		fmt.Printf("Skipping unsupported file: %s\n", target)
		return nil
	}
}
