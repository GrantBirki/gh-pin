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
