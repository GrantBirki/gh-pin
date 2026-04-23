package processor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/regclient/regclient"
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

func TestProcessComposeContent_RewritesImagesAndPreservesSuffix(t *testing.T) {
	config := ProcessorConfig{
		Algorithm: "sha256",
		ImagePinner: func(_ *regclient.RegClient, image string, _ ProcessorConfig) (string, error) {
			return image + "@sha256:abc123", nil
		},
	}

	input := []byte(`services:
  web:
    image: nginx:alpine # keep this comment
  worker:
    image: busybox:latest
`)
	output, changed, err := processComposeContent(nil, input, config)
	if err != nil {
		t.Fatalf("processComposeContent() error: %v", err)
	}
	if !changed {
		t.Fatal("processComposeContent() changed = false, want true")
	}

	want := `services:
  web:
    image: nginx:alpine@sha256:abc123 # keep this comment
  worker:
    image: busybox:latest@sha256:abc123
`
	if string(output) != want {
		t.Fatalf("output = %q, want %q", string(output), want)
	}
}

func TestProcessComposeContent_PinErrorLeavesImageUnchanged(t *testing.T) {
	config := ProcessorConfig{
		Algorithm: "sha256",
		ImagePinner: func(_ *regclient.RegClient, _ string, _ ProcessorConfig) (string, error) {
			return "", errors.New("registry unavailable")
		},
	}

	input := []byte("services:\n  web:\n    image: nginx:alpine\n")
	output, changed, err := processComposeContent(nil, input, config)
	if err != nil {
		t.Fatalf("processComposeContent() error: %v", err)
	}
	if changed {
		t.Fatal("processComposeContent() changed = true, want false")
	}
	if string(output) != string(input) {
		t.Fatalf("output = %q, want original %q", string(output), string(input))
	}
}
