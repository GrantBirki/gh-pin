package processor

// ComposeFile is a minimal representation of a docker-compose YAML
type ComposeFile struct {
	Services map[string]struct {
		Image string `yaml:"image"`
	} `yaml:"services"`
}

// ProcessorConfig holds configuration for the processor
type ProcessorConfig struct {
	DryRun    bool
	Algorithm string
	NoColor   bool
}
