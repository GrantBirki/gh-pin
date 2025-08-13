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

// isGitHubWorkflowFile checks if a file is a GitHub Actions workflow file
func isGitHubWorkflowFile(path string) bool {
	// Check if file is in .github/workflows/ directory
	if !strings.Contains(path, ".github/workflows/") {
		return false
	}
	
	// Check if it's a YAML file
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yml" || ext == ".yaml"
}

// detectFileType analyzes a YAML file to determine its type
func detectFileType(path string, data []byte) string {
	// First check if it's a GitHub Actions workflow by path
	if isGitHubWorkflowFile(path) {
		return "actions"
	}
	
	// Check if it's a Docker Compose file by structure
	var cf processor.ComposeFile
	if err := yaml.Unmarshal(data, &cf); err == nil && len(cf.Services) > 0 {
		return "compose"
	}
	
	// Check if it contains GitHub Actions workflow structure
	var workflow map[string]interface{}
	if err := yaml.Unmarshal(data, &workflow); err == nil {
		if _, hasJobs := workflow["jobs"]; hasJobs {
			if _, hasOn := workflow["on"]; hasOn {
				return "actions"
			}
		}
	}
	
	return "unknown"
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
			lower == "docker-compose.yaml":
			fmt.Printf("Processing Compose: %s\n", path)
			return processor.ProcessCompose(rc, path, config)
		case isGitHubWorkflowFile(path):
			fmt.Printf("Processing GitHub Actions: %s\n", path)
			return processor.ProcessActions(rc, path, config)
		case config.Pervasive && (ext == ".yml" || ext == ".yaml"):
			// Only process generic YAML files when --pervasive flag is used
			// Detect the file type and process accordingly
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			
			fileType := detectFileType(path, data)
			switch fileType {
			case "compose":
				fmt.Printf("Processing Compose: %s\n", path)
				return processor.ProcessCompose(rc, path, config)
			case "actions":
				fmt.Printf("Processing GitHub Actions: %s\n", path)
				return processor.ProcessActions(rc, path, config)
			default:
				// Skip unknown YAML files
				return nil
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
		// For explicitly specified files, detect the type
		data, err := os.ReadFile(target)
		if err != nil {
			return err
		}
		
		fileType := detectFileType(target, data)
		switch fileType {
		case "compose":
			fmt.Printf("Processing Compose: %s\n", target)
			return processor.ProcessCompose(rc, target, config)
		case "actions":
			fmt.Printf("Processing GitHub Actions: %s\n", target)
			return processor.ProcessActions(rc, target, config)
		default:
			// Fall back to trying compose processing for backward compatibility
			fmt.Printf("Processing as Compose: %s\n", target)
			return processor.ProcessCompose(rc, target, config)
		}
	default:
		fmt.Printf("Skipping unsupported file: %s\n", target)
		return nil
	}
}
