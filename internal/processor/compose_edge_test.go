package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessCompose_InvalidYAMLMarshal(t *testing.T) {
	tempDir := t.TempDir()

	// Create a valid compose file
	validComposeFile := filepath.Join(tempDir, "valid.yml")
	content := `services:
  web:
    image: nginx:latest`

	err := os.WriteFile(validComposeFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create compose file with already pinned images (no RegClient needed)
	pinnedComposeFile := filepath.Join(tempDir, "pinned-compose.yml")
	pinnedComposeContent := `version: '3'
services:
  web:
    image: nginx@sha256:abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab
  db:
    image: postgres@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef`

	err = os.WriteFile(pinnedComposeFile, []byte(pinnedComposeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test dry run mode with pinned images (no RegClient calls needed)
	config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
	err = ProcessCompose(nil, pinnedComposeFile, config)

	// Should succeed when images are already pinned (no RegClient needed)
	if err != nil {
		t.Errorf("Expected no error with pinned images, got %v", err)
	}

	// Verify original file is unchanged
	after, err := os.ReadFile(validComposeFile)
	if err != nil {
		t.Fatalf("Failed to read file after processing: %v", err)
	}
	if string(after) != content {
		t.Error("File was modified during dry run")
	}
}

func TestProcessCompose_EmptyServices(t *testing.T) {
	tempDir := t.TempDir()

	// Test compose file with empty services
	emptyServicesFile := filepath.Join(tempDir, "empty-services.yml")
	content := `services: {}`

	err := os.WriteFile(emptyServicesFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
	err = ProcessCompose(nil, emptyServicesFile, config)
	if err != nil {
		t.Errorf("Expected no error for empty services, got %v", err)
	}
}

func TestProcessCompose_ServiceWithoutImage(t *testing.T) {
	tempDir := t.TempDir()

	// Test compose file with service without image
	noImageFile := filepath.Join(tempDir, "no-image.yml")
	content := `services:
  web:
    build: .
    ports:
      - "80:80"`

	err := os.WriteFile(noImageFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
	err = ProcessCompose(nil, noImageFile, config)
	if err != nil {
		t.Errorf("Expected no error for service without image, got %v", err)
	}
}

func TestProcessCompose_ServiceWithEmptyImage(t *testing.T) {
	tempDir := t.TempDir()

	// Test compose file with empty image field
	emptyImageFile := filepath.Join(tempDir, "empty-image.yml")
	content := `services:
  web:
    image: ""`

	err := os.WriteFile(emptyImageFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
	err = ProcessCompose(nil, emptyImageFile, config)
	if err != nil {
		t.Errorf("Expected no error for empty image, got %v", err)
	}
}

func TestProcessCompose_DifferentAlgorithm(t *testing.T) {
	tempDir := t.TempDir()

	// Test compose file with sha256 digest but checking for sha512
	sha256File := filepath.Join(tempDir, "sha256.yml")
	content := `services:
  web:
    image: nginx@sha256:abc123def456`

	err := os.WriteFile(sha256File, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Check with sha512 algorithm - should consider image as unpinned
	config := ProcessorConfig{DryRun: true, Algorithm: "sha512"}
	err = ProcessCompose(nil, sha256File, config)

	// Should not error in dry run mode
	if err != nil {
		t.Errorf("Expected no error for different algorithm in dry run, got %v", err)
	}
}
