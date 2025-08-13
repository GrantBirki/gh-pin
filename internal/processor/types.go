package processor

import "os"

// ComposeFile is a flexible representation of a docker-compose YAML that preserves all fields
type ComposeFile map[string]interface{}

// ProcessorConfig holds configuration for the processor
type ProcessorConfig struct {
	DryRun         bool
	Algorithm      string
	NoColor        bool
	Pervasive      bool
	ExpandRegistry bool
	ForceMode      string         // "docker", "actions", or "" for auto-detection
	Quiet          bool           // suppress informational messages when no changes are needed
	Platform       string         // platform-specific manifest digest (e.g., linux/amd64, linux/arm/v7)
	GitHubResolver GitHubResolver // resolver for GitHub API calls (injected for testing)
}

// getFileMode returns the file mode of the given path, defaulting to 0644 if unable to stat
func getFileMode(path string) os.FileMode {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().Perm()
	}
	return 0644 // fallback to default permissions
}
