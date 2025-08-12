package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessDockerfile_FileErrors(t *testing.T) {
	tempDir := t.TempDir()

	// Test non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "nonexistent")
		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}

		err := ProcessDockerfile(nil, nonExistentFile, config)
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})

	// Test file with no FROM statements
	t.Run("no from statements", func(t *testing.T) {
		dockerfileNoFrom := filepath.Join(tempDir, "Dockerfile.nofrom")
		content := `# Comment only
RUN echo "hello"
COPY . /app
WORKDIR /app`
		err := os.WriteFile(dockerfileNoFrom, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
		err = ProcessDockerfile(nil, dockerfileNoFrom, config)
		if err != nil {
			t.Errorf("Expected no error for Dockerfile without FROM statements, got %v", err)
		}

		// Verify file wasn't modified
		after, err := os.ReadFile(dockerfileNoFrom)
		if err != nil {
			t.Fatalf("Failed to read file after processing: %v", err)
		}
		if string(after) != content {
			t.Error("File was modified during dry run")
		}
	})

	// Test file with already pinned FROM statements (dry run)
	t.Run("already pinned from statements dry run", func(t *testing.T) {
		dockerfilePinned := filepath.Join(tempDir, "Dockerfile.pinned")
		content := `FROM nginx@sha256:abc123def456
FROM postgres@sha256:def456ghi789
RUN echo "test"`
		err := os.WriteFile(dockerfilePinned, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
		err = ProcessDockerfile(nil, dockerfilePinned, config)
		if err != nil {
			t.Errorf("Expected no error for already pinned FROM statements, got %v", err)
		}

		// Verify file wasn't modified
		after, err := os.ReadFile(dockerfilePinned)
		if err != nil {
			t.Fatalf("Failed to read file after processing: %v", err)
		}
		if string(after) != content {
			t.Error("File was modified during dry run")
		}
	})

	// Test file with mixed case FROM statements and indentation
	t.Run("mixed case and indentation", func(t *testing.T) {
		dockerfileMixed := filepath.Join(tempDir, "Dockerfile.mixed")
		content := `# Base image
FROM ubuntu@sha256:abc123
  from nginx@sha256:def456
    FROM postgres@sha256:ghi789
RUN echo "test"`
		err := os.WriteFile(dockerfileMixed, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
		err = ProcessDockerfile(nil, dockerfileMixed, config)
		if err != nil {
			t.Errorf("Expected no error for mixed case FROM statements, got %v", err)
		}
	})

	// Test file with FROM in comments (should be ignored)
	t.Run("from in comments", func(t *testing.T) {
		dockerfileComments := filepath.Join(tempDir, "Dockerfile.comments")
		content := `# FROM ubuntu:latest should be ignored
FROM nginx@sha256:abc123
# Another comment with FROM postgres:13
RUN echo "test"`
		err := os.WriteFile(dockerfileComments, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
		err = ProcessDockerfile(nil, dockerfileComments, config)
		if err != nil {
			t.Errorf("Expected no error for Dockerfile with FROM in comments, got %v", err)
		}

		// Verify file wasn't modified (FROM is already pinned)
		after, err := os.ReadFile(dockerfileComments)
		if err != nil {
			t.Fatalf("Failed to read file after processing: %v", err)
		}
		if string(after) != content {
			t.Error("File was modified during dry run")
		}
	})

	// Test malformed FROM statements
	t.Run("malformed from statements", func(t *testing.T) {
		dockerfileMalformed := filepath.Join(tempDir, "Dockerfile.malformed")
		content := `FROM
FROM 
FROM   
FROM ubuntu@sha256:abc123`
		err := os.WriteFile(dockerfileMalformed, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		config := ProcessorConfig{DryRun: true, Algorithm: "sha256"}
		err = ProcessDockerfile(nil, dockerfileMalformed, config)
		if err != nil {
			t.Errorf("Expected no error for malformed FROM statements, got %v", err)
		}
	})
}

func TestDockerfileLineProcessing(t *testing.T) {
	// Test different FROM line formats that should be recognized
	testLines := []struct {
		name        string
		line        string
		shouldMatch bool
	}{
		{"standard FROM", "FROM ubuntu:latest", true},
		{"uppercase FROM", "FROM UBUNTU:LATEST", true},
		{"lowercase from", "from nginx:alpine", true},
		{"mixed case from", "From postgres:13", true},
		{"indented FROM", "  FROM redis:6", true},
		{"tab indented FROM", "\tFROM mongo:4", true},
		{"FROM with extra spaces", "FROM   alpine:3.14", true},
		{"FROM in comment", "# FROM ubuntu:latest", false},
		{"partial FROM", "FROMS ubuntu:latest", false},
		{"FROM substring", "DOCKERFILE FROM ubuntu", false},
		{"RUN with FROM", "RUN echo FROM", false},
	}

	for _, tt := range testLines {
		t.Run(tt.name, func(t *testing.T) {
			trim := strings.TrimSpace(tt.line)
			isFromLine := strings.HasPrefix(strings.ToUpper(trim), "FROM ")

			if isFromLine != tt.shouldMatch {
				t.Errorf("Line %q: expected match %v, got %v", tt.line, tt.shouldMatch, isFromLine)
			}
		})
	}
}
