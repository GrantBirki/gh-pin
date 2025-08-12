package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessDockerfile_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()

	// Test Dockerfile with only comments and empty lines
	t.Run("comments and empty lines only", func(t *testing.T) {
		dockerfileComments := filepath.Join(tempDir, "Dockerfile.comments")
		content := `# This is a comment
		
		# Another comment
		
		# FROM is in a comment
		`
		err := os.WriteFile(dockerfileComments, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
		err = ProcessDockerfile(nil, dockerfileComments, config)
		if err != nil {
			t.Errorf("Expected no error for comments-only Dockerfile, got %v", err)
		}
	})

	// Test Dockerfile with FROM in the middle of a line
	t.Run("FROM not at start of trimmed line", func(t *testing.T) {
		dockerfileMiddle := filepath.Join(tempDir, "Dockerfile.middle")
		content := `RUN echo "FROM ubuntu:latest"
COPY FROM_file /app/
ENV MESSAGE="FROM ubuntu we copy"`
		err := os.WriteFile(dockerfileMiddle, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
		err = ProcessDockerfile(nil, dockerfileMiddle, config)
		if err != nil {
			t.Errorf("Expected no error for Dockerfile with FROM in middle of lines, got %v", err)
		}
	})

	// Test Dockerfile with complex FROM statements
	t.Run("complex FROM statements", func(t *testing.T) {
		dockerfileComplex := filepath.Join(tempDir, "Dockerfile.complex")
		content := `FROM ubuntu@sha256:abc123 AS builder
FROM --platform=linux/amd64 nginx@sha256:def456
FROM registry.example.com/myimage@sha256:ghi789`
		err := os.WriteFile(dockerfileComplex, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
		err = ProcessDockerfile(nil, dockerfileComplex, config)
		if err != nil {
			t.Errorf("Expected no error for complex FROM statements, got %v", err)
		}
	})

	// Test Dockerfile with different digest algorithms
	t.Run("different algorithm check", func(t *testing.T) {
		dockerfileSha256 := filepath.Join(tempDir, "Dockerfile.sha256")
		content := `FROM ubuntu@sha256:abc123def456`
		err := os.WriteFile(dockerfileSha256, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Check with sha512 algorithm - should consider as unpinned
		config := ProcessorConfig{DryRun: true, Algorithm: "sha512"}
		err = ProcessDockerfile(nil, dockerfileSha256, config)
		if err != nil {
			t.Errorf("Expected no error for different algorithm check, got %v", err)
		}
	})

	// Test preserving indentation
	t.Run("indentation preservation", func(t *testing.T) {
		dockerfileIndent := filepath.Join(tempDir, "Dockerfile.indent")
		content := `# Test indentation
		FROM ubuntu@sha256:abc123
	FROM nginx@sha256:def456
    FROM postgres@sha256:ghi789`
		err := os.WriteFile(dockerfileIndent, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
		err = ProcessDockerfile(nil, dockerfileIndent, config)
		if err != nil {
			t.Errorf("Expected no error for indented Dockerfile, got %v", err)
		}

		// Verify content unchanged in dry run
		after, err := os.ReadFile(dockerfileIndent)
		if err != nil {
			t.Fatalf("Failed to read file after processing: %v", err)
		}
		if string(after) != content {
			t.Error("File was modified during dry run")
		}
	})
}

func TestProcessDockerfile_ScannerError(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file that might cause scanner issues (though standard scanner is quite robust)
	// This test is mainly for coverage of the scanner.Err() check
	largeDockerfile := filepath.Join(tempDir, "Dockerfile.large")

	// Create a large Dockerfile to test scanner behavior
	var content strings.Builder
	for i := 0; i < 1000; i++ {
		content.WriteString("# Comment line number ")
		content.WriteString(strings.Repeat("x", 100))
		content.WriteString("\n")
	}
	content.WriteString("FROM ubuntu@sha256:abc123\n")

	err := os.WriteFile(largeDockerfile, []byte(content.String()), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
	err = ProcessDockerfile(nil, largeDockerfile, config)
	if err != nil {
		t.Errorf("Expected no error for large Dockerfile, got %v", err)
	}
}

func TestProcessDockerfile_WritePermissions(t *testing.T) {
	tempDir := t.TempDir()

	// Test file permissions are preserved during dry run
	dockerfilePerms := filepath.Join(tempDir, "Dockerfile.perms")
	content := `FROM ubuntu@sha256:abc123`

	err := os.WriteFile(dockerfilePerms, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify initial permissions
	info, err := os.Stat(dockerfilePerms)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	initialMode := info.Mode().Perm()

	config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
	err = ProcessDockerfile(nil, dockerfilePerms, config)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify permissions unchanged
	info, err = os.Stat(dockerfilePerms)
	if err != nil {
		t.Fatalf("Failed to stat file after processing: %v", err)
	}
	if info.Mode().Perm() != initialMode {
		t.Errorf("File permissions changed: expected %v, got %v", initialMode, info.Mode().Perm())
	}
}
