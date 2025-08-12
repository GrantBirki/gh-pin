package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessCompose_FileErrors(t *testing.T) {
	tempDir := t.TempDir()

	// Test non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "nonexistent.yml")
		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}

		err := ProcessCompose(nil, nonExistentFile, config)
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})

	// Test invalid YAML
	t.Run("invalid yaml", func(t *testing.T) {
		invalidYamlFile := filepath.Join(tempDir, "invalid.yml")
		err := os.WriteFile(invalidYamlFile, []byte("invalid: yaml: content: ["), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
		err = ProcessCompose(nil, invalidYamlFile, config)
		if err == nil {
			t.Error("Expected error for invalid YAML, got nil")
		}
	})

	// Test valid YAML but no services
	t.Run("no services", func(t *testing.T) {
		noServicesFile := filepath.Join(tempDir, "no-services.yml")
		content := `version: "3.8"
networks:
  default:`
		err := os.WriteFile(noServicesFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
		err = ProcessCompose(nil, noServicesFile, config)
		if err != nil {
			t.Errorf("Expected no error for valid YAML without services, got %v", err)
		}
	})

	// Test compose with already pinned images (dry run)
	t.Run("already pinned images dry run", func(t *testing.T) {
		pinnedFile := filepath.Join(tempDir, "pinned.yml")
		content := `services:
  web:
    image: nginx@sha256:abc123def456
  db:
    image: postgres@sha256:def456ghi789`
		err := os.WriteFile(pinnedFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
		err = ProcessCompose(nil, pinnedFile, config)
		if err != nil {
			t.Errorf("Expected no error for already pinned images, got %v", err)
		}

		// Verify file wasn't modified
		after, err := os.ReadFile(pinnedFile)
		if err != nil {
			t.Fatalf("Failed to read file after processing: %v", err)
		}
		if string(after) != content {
			t.Error("File was modified during dry run")
		}
	})
}
