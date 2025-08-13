package processor

import "os"

// ComposeFile is a minimal representation of a docker-compose YAML
type ComposeFile struct {
	Services map[string]struct {
		Image string `yaml:"image"`
	} `yaml:"services"`
}

// ProcessorConfig holds configuration for the processor
type ProcessorConfig struct {
	DryRun         bool
	Algorithm      string
	NoColor        bool
	Pervasive      bool
	ExpandRegistry bool
	ForceMode      string // "docker", "actions", or "" for auto-detection
}

// getFileMode returns the file mode of the given path, defaulting to 0644 if unable to stat
func getFileMode(path string) os.FileMode {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().Perm()
	}
	return 0644 // fallback to default permissions
}
