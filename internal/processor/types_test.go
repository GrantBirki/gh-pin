package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetFileMode(t *testing.T) {
	tests := []struct {
		name          string
		setupFile     bool
		fileMode      os.FileMode
		expectedMode  os.FileMode
		expectDefault bool
	}{
		{
			name:          "existing file with 644 permissions",
			setupFile:     true,
			fileMode:      0644,
			expectedMode:  0644,
			expectDefault: false,
		},
		{
			name:          "existing file with 755 permissions",
			setupFile:     true,
			fileMode:      0755,
			expectedMode:  0755,
			expectDefault: false,
		},
		{
			name:          "existing file with 600 permissions",
			setupFile:     true,
			fileMode:      0600,
			expectedMode:  0600,
			expectDefault: false,
		},
		{
			name:          "non-existent file returns default",
			setupFile:     false,
			expectedMode:  0644,
			expectDefault: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "testfile")

			if tt.setupFile {
				// Create file with specific permissions
				err := os.WriteFile(testFile, []byte("test content"), tt.fileMode)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			} else {
				// Use a non-existent file path
				testFile = filepath.Join(tempDir, "nonexistent")
			}

			result := getFileMode(testFile)
			if result != tt.expectedMode {
				t.Errorf("Expected mode %v, got %v", tt.expectedMode, result)
			}
		})
	}
}

func TestProcessorConfig(t *testing.T) {
	// Test ProcessorConfig struct creation and field access
	config := ProcessorConfig{
		DryRun:    true,
		Algorithm: "sha256",
		NoColor:   false,
	}

	if !config.DryRun {
		t.Errorf("Expected DryRun to be true, got %v", config.DryRun)
	}
	if config.Algorithm != "sha256" {
		t.Errorf("Expected Algorithm to be 'sha256', got %v", config.Algorithm)
	}
	if config.NoColor {
		t.Errorf("Expected NoColor to be false, got %v", config.NoColor)
	}
}

func TestComposeFile(t *testing.T) {
	// Test ComposeFile struct creation and field access
	cf := ComposeFile{
		Services: map[string]struct {
			Image string `yaml:"image"`
		}{
			"web": {Image: "nginx:latest"},
			"db":  {Image: "postgres:13"},
		},
	}

	if len(cf.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(cf.Services))
	}
	if cf.Services["web"].Image != "nginx:latest" {
		t.Errorf("Expected web service image to be 'nginx:latest', got %v", cf.Services["web"].Image)
	}
	if cf.Services["db"].Image != "postgres:13" {
		t.Errorf("Expected db service image to be 'postgres:13', got %v", cf.Services["db"].Image)
	}
}
