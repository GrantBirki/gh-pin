package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grantbirki/gh-pin/internal/processor"
)

func TestIsStandardDockerfile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		// Standard Dockerfile names (should match)
		{"exact Dockerfile", "Dockerfile", true},
		{"uppercase Dockerfile", "DOCKERFILE", true},
		{"mixed case Dockerfile", "DockerFile", true},
		{"Dockerfile with .dockerfile extension", "app.dockerfile", true},
		{"uppercase .dockerfile", "APP.DOCKERFILE", true},
		{"Dockerfile with stage suffix", "Dockerfile.dev", true},
		{"Dockerfile with prod suffix", "Dockerfile.prod", true},
		{"Dockerfile with production suffix", "Dockerfile.production", true},
		{"Dockerfile with test suffix", "Dockerfile.test", true},
		{"Dockerfile with staging suffix", "Dockerfile.staging", true},
		{"Dockerfile with local suffix", "Dockerfile.local", true},
		{"Dockerfile with build suffix", "Dockerfile.build", true},

		// Non-standard names (should not match)
		{"Dockerfile with dash", "Dockerfile-temp", false},
		{"Dockerfile with underscore", "Dockerfile_old", false},
		{"Dockerfile backup", "Dockerfile.backup", true}, // This actually matches due to Dockerfile.* pattern
		{"Dockerfile with dash suffix", "Dockerfile-dev", false},
		{"File starting with Dockerfile but not standard", "Dockerfile123", false},
		{"Random file", "README.md", false},
		{"Empty filename", "", false},
		{"Just extension", ".dockerfile", true},                               // This matches due to .dockerfile suffix
		{"File ending in dockerfile but not starting", "my.dockerfile", true}, // This one should match due to .dockerfile extension
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStandardDockerfile(tt.filename)
			if result != tt.expected {
				t.Errorf("isStandardDockerfile(%q) = %v, expected %v",
					tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsGitHubWorkflowFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "workflow yml",
			path: "/repo/.github/workflows/ci.yml",
			want: true,
		},
		{
			name: "workflow yaml",
			path: "/repo/.github/workflows/release.yaml",
			want: true,
		},
		{
			name: "workflow directory but not yaml",
			path: "/repo/.github/workflows/README.md",
			want: false,
		},
		{
			name: "yaml outside workflow directory",
			path: "/repo/config.yml",
			want: false,
		},
		{
			name: "similarly named directory",
			path: "/repo/.github/workflows-old/ci.yml",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGitHubWorkflowFile(tt.path)
			if got != tt.want {
				t.Fatalf("isGitHubWorkflowFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDetectFileType(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		data   string
		config processor.ProcessorConfig
		want   string
	}{
		{
			name: "actions by workflow path",
			path: "/repo/.github/workflows/ci.yml",
			data: "not: workflow",
			want: "actions",
		},
		{
			name: "compose by services structure",
			path: "/repo/docker-compose.yml",
			data: "services:\n  web:\n    image: nginx:latest\n",
			want: "compose",
		},
		{
			name: "actions by yaml structure",
			path: "/repo/build.yml",
			data: "on: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n",
			want: "actions",
		},
		{
			name: "unknown yaml",
			path: "/repo/values.yaml",
			data: "database:\n  host: localhost\n",
			want: "unknown",
		},
		{
			name:   "force docker accepts compose only",
			path:   "/repo/config.yml",
			data:   "services:\n  web:\n    image: nginx:latest\n",
			config: processor.ProcessorConfig{ForceMode: "docker"},
			want:   "compose",
		},
		{
			name:   "force docker rejects actions",
			path:   "/repo/.github/workflows/ci.yml",
			data:   "on: push\njobs:\n  test: {}\n",
			config: processor.ProcessorConfig{ForceMode: "docker"},
			want:   "unknown",
		},
		{
			name:   "force actions accepts workflow path",
			path:   "/repo/.github/workflows/ci.yaml",
			data:   "services:\n  web:\n    image: nginx:latest\n",
			config: processor.ProcessorConfig{ForceMode: "actions"},
			want:   "actions",
		},
		{
			name:   "force actions accepts workflow shape",
			path:   "/repo/config.yml",
			data:   "on: pull_request\njobs:\n  test: {}\n",
			config: processor.ProcessorConfig{ForceMode: "actions"},
			want:   "actions",
		},
		{
			name:   "force actions rejects compose",
			path:   "/repo/docker-compose.yml",
			data:   "services:\n  web:\n    image: nginx:latest\n",
			config: processor.ProcessorConfig{ForceMode: "actions"},
			want:   "unknown",
		},
		{
			name: "invalid yaml",
			path: "/repo/broken.yml",
			data: "invalid: yaml: [",
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectFileType(tt.path, []byte(tt.data), tt.config)
			if got != tt.want {
				t.Fatalf("detectFileType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScanPath_FileSystemErrors(t *testing.T) {
	tempDir := t.TempDir()
	config := processor.ProcessorConfig{DryRun: true, Algorithm: "sha256"}

	// Test non-existent directory
	t.Run("non-existent directory", func(t *testing.T) {
		nonExistentDir := filepath.Join(tempDir, "nonexistent")
		err := ScanPath(nil, nonExistentDir, config, false)
		if err == nil {
			t.Error("Expected error for non-existent directory, got nil")
		}
	})

	// Test empty directory
	t.Run("empty directory", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.Mkdir(emptyDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create empty directory: %v", err)
		}

		err = ScanPath(nil, emptyDir, config, false)
		if err != nil {
			t.Errorf("Expected no error for empty directory, got %v", err)
		}
	})

	// Test directory with non-matching files
	t.Run("directory with non-matching files", func(t *testing.T) {
		nonMatchingDir := filepath.Join(tempDir, "nonmatching")
		err := os.Mkdir(nonMatchingDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Create some non-matching files
		files := []string{"README.md", "script.sh", "config.json"}
		for _, file := range files {
			err := os.WriteFile(filepath.Join(nonMatchingDir, file), []byte("test"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file %s: %v", file, err)
			}
		}

		err = ScanPath(nil, nonMatchingDir, config, false)
		if err != nil {
			t.Errorf("Expected no error for directory with non-matching files, got %v", err)
		}
	})

	// Test recursive vs non-recursive scanning
	t.Run("recursive vs non-recursive", func(t *testing.T) {
		baseDir := filepath.Join(tempDir, "recursive-test")
		err := os.Mkdir(baseDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create base directory: %v", err)
		}

		subDir := filepath.Join(baseDir, "subdir")
		err = os.Mkdir(subDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}

		// Create a Dockerfile in subdirectory
		dockerfilePath := filepath.Join(subDir, "Dockerfile")
		content := "FROM nginx@sha256:abc123"
		err = os.WriteFile(dockerfilePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create Dockerfile: %v", err)
		}

		// Test non-recursive (should not find the Dockerfile in subdir)
		err = ScanPath(nil, baseDir, config, false)
		if err != nil {
			t.Errorf("Expected no error for non-recursive scan, got %v", err)
		}

		// Test recursive (would find the Dockerfile but we're using dry run with already pinned image)
		err = ScanPath(nil, baseDir, config, true)
		if err != nil {
			t.Errorf("Expected no error for recursive scan, got %v", err)
		}
	})
}

func TestScanPath_ForceModes(t *testing.T) {
	tempDir := t.TempDir()
	workflowDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll() error: %v", err)
	}

	files := map[string]string{
		"Dockerfile":         ".dockerignore only\n",
		"docker-compose.yml": "services:\n  web:\n    build: .\n",
		filepath.Join(".github", "workflows", "ci.yml"): "steps:\n  - uses: actions/checkout@0123456789012345678901234567890123456789\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("os.WriteFile(%q) error: %v", name, err)
		}
	}

	tests := []struct {
		name   string
		config processor.ProcessorConfig
	}{
		{
			name:   "docker only skips actions",
			config: processor.ProcessorConfig{DryRun: true, Algorithm: "sha256", ForceMode: "docker"},
		},
		{
			name:   "actions only skips docker files",
			config: processor.ProcessorConfig{DryRun: true, Algorithm: "sha256", ForceMode: "actions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ScanPath(nil, tempDir, tt.config, true); err != nil {
				t.Fatalf("ScanPath() error: %v", err)
			}
		})
	}
}

func TestProcessSingleFile_FileTypes(t *testing.T) {
	tempDir := t.TempDir()
	config := processor.ProcessorConfig{DryRun: true, Algorithm: "sha256"}

	tests := []struct {
		name        string
		filename    string
		content     string
		expectError bool
	}{
		{
			name:        "standard Dockerfile",
			filename:    "Dockerfile",
			content:     "FROM nginx@sha256:abc123",
			expectError: false,
		},
		{
			name:        "Dockerfile with dash",
			filename:    "Dockerfile-temp",
			content:     "FROM nginx@sha256:abc123",
			expectError: false, // ProcessSingleFile is more permissive
		},
		{
			name:        "docker-compose.yml",
			filename:    "docker-compose.yml",
			content:     "services:\n  web:\n    image: nginx@sha256:abc123",
			expectError: false,
		},
		{
			name:        "generic yaml file",
			filename:    "config.yml",
			content:     "services:\n  web:\n    image: nginx@sha256:abc123",
			expectError: false,
		},
		{
			name:        "unsupported file",
			filename:    "README.md",
			content:     "# This is a readme",
			expectError: false, // No error, just skipped
		},
		{
			name:        "non-existent file",
			filename:    "nonexistent.txt",
			content:     "",    // Won't create this file
			expectError: false, // ProcessSingleFile doesn't error on non-existent files, just skips them
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string
			if tt.filename == "nonexistent.txt" {
				filePath = filepath.Join(tempDir, tt.filename)
			} else {
				filePath = filepath.Join(tempDir, tt.filename)
				err := os.WriteFile(filePath, []byte(tt.content), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			err := ProcessSingleFile(nil, filePath, config)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestProcessSingleFile_ForceModeSkips(t *testing.T) {
	tempDir := t.TempDir()

	composePath := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte("services:\n  web:\n    image: nginx@sha256:abc123\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error: %v", err)
	}

	workflowPath := filepath.Join(tempDir, "workflow.yml")
	if err := os.WriteFile(workflowPath, []byte("on: push\njobs:\n  test: {}\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error: %v", err)
	}

	dockerfilePath := filepath.Join(tempDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte("FROM nginx@sha256:abc123\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error: %v", err)
	}

	tests := []struct {
		name   string
		target string
		config processor.ProcessorConfig
	}{
		{
			name:   "compose skipped in actions mode",
			target: composePath,
			config: processor.ProcessorConfig{DryRun: true, Algorithm: "sha256", ForceMode: "actions"},
		},
		{
			name:   "workflow skipped in docker mode",
			target: workflowPath,
			config: processor.ProcessorConfig{DryRun: true, Algorithm: "sha256", ForceMode: "docker"},
		},
		{
			name:   "dockerfile skipped in actions mode",
			target: dockerfilePath,
			config: processor.ProcessorConfig{DryRun: true, Algorithm: "sha256", ForceMode: "actions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ProcessSingleFile(nil, tt.target, tt.config); err != nil {
				t.Fatalf("ProcessSingleFile() error: %v", err)
			}
		})
	}
}

func TestProcessSingleFile_ActionsAndUnknownYaml(t *testing.T) {
	tempDir := t.TempDir()

	workflowDir := filepath.Join(tempDir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll() error: %v", err)
	}

	workflowPath := filepath.Join(workflowDir, "ci.yml")
	if err := os.WriteFile(workflowPath, []byte("steps:\n  - uses: actions/checkout@0123456789012345678901234567890123456789\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error: %v", err)
	}

	unknownYamlPath := filepath.Join(tempDir, "values.yml")
	if err := os.WriteFile(unknownYamlPath, []byte("database:\n  host: localhost\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error: %v", err)
	}

	if err := ProcessSingleFile(nil, workflowPath, processor.ProcessorConfig{DryRun: true, Algorithm: "sha256"}); err != nil {
		t.Fatalf("ProcessSingleFile(workflow) error: %v", err)
	}
	if err := ProcessSingleFile(nil, unknownYamlPath, processor.ProcessorConfig{DryRun: true, Algorithm: "sha256"}); err != nil {
		t.Fatalf("ProcessSingleFile(unknown yaml) error: %v", err)
	}
}

func TestProcessSingleFile_FilenameDetection(t *testing.T) {
	tempDir := t.TempDir()
	config := processor.ProcessorConfig{DryRun: true, Algorithm: "sha256"}

	// Test various Dockerfile naming patterns that should be processed
	dockerfileNames := []string{
		"Dockerfile",
		"DOCKERFILE",
		"dockerfile",
		"Dockerfile.dev",
		"Dockerfile-temp", // Should work with ProcessSingleFile
		"Dockerfile_old",  // Should work with ProcessSingleFile
		"app.dockerfile",
		"build.dockerfile",
	}

	for _, name := range dockerfileNames {
		t.Run("dockerfile_"+name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, name)
			content := "FROM nginx@sha256:abc123"
			err := os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			err = ProcessSingleFile(nil, filePath, config)
			if err != nil {
				t.Errorf("Expected no error for Dockerfile %q, got %v", name, err)
			}
		})
	}

	// Test YAML files
	yamlNames := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"config.yml",
		"values.yaml",
	}

	for _, name := range yamlNames {
		t.Run("yaml_"+name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, name)
			content := "services:\n  web:\n    image: nginx@sha256:abc123"
			err := os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			err = ProcessSingleFile(nil, filePath, config)
			if err != nil {
				t.Errorf("Expected no error for YAML %q, got %v", name, err)
			}
		})
	}
}

func TestScanPath_PervasiveFlag(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"docker-compose.yml": "services:\n  web:\n    image: nginx@sha256:abc123",
		"config.yml":         "services:\n  web:\n    image: nginx@sha256:def456", // Generic YAML with services
		"values.yaml":        "database:\n  host: localhost\nport: 5432",          // Generic YAML without services
		"Dockerfile":         "FROM nginx@sha256:abc123",
	}

	for filename, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	t.Run("without pervasive flag", func(t *testing.T) {
		config := processor.ProcessorConfig{DryRun: true, Algorithm: "sha256", Pervasive: false}
		err := ScanPath(nil, tempDir, config, false)
		if err != nil {
			t.Errorf("Expected no error without pervasive flag, got %v", err)
		}
		// Should only process docker-compose.yml and Dockerfile, not config.yml
	})

	t.Run("with pervasive flag", func(t *testing.T) {
		config := processor.ProcessorConfig{DryRun: true, Algorithm: "sha256", Pervasive: true}
		err := ScanPath(nil, tempDir, config, false)
		if err != nil {
			t.Errorf("Expected no error with pervasive flag, got %v", err)
		}
		// Should process docker-compose.yml, config.yml (has services), and Dockerfile
		// Should skip values.yaml (no services section)
	})
}
