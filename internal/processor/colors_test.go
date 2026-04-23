package processor

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestParseImageNameAndTag(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		wantName string
		wantTag  string
	}{
		{
			name:     "explicit tag",
			image:    "nginx:alpine",
			wantName: "nginx",
			wantTag:  "alpine",
		},
		{
			name:     "no tag defaults latest",
			image:    "ubuntu",
			wantName: "ubuntu",
			wantTag:  "latest",
		},
		{
			name:     "fully qualified image",
			image:    "ghcr.io/owner/image:v1.2.3",
			wantName: "ghcr.io/owner/image",
			wantTag:  "v1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotTag := ParseImageNameAndTag(tt.image)
			if gotName != tt.wantName || gotTag != tt.wantTag {
				t.Fatalf("ParseImageNameAndTag(%q) = (%q, %q), want (%q, %q)",
					tt.image, gotName, gotTag, tt.wantName, tt.wantTag)
			}
		})
	}
}

func TestParseImageNameAndDigest(t *testing.T) {
	tests := []struct {
		name       string
		image      string
		wantName   string
		wantDigest string
	}{
		{
			name:       "digest reference",
			image:      "nginx:alpine@sha256:abc123",
			wantName:   "nginx:alpine",
			wantDigest: "sha256:abc123",
		},
		{
			name:       "no digest",
			image:      "nginx:alpine",
			wantName:   "nginx:alpine",
			wantDigest: "",
		},
		{
			name:       "multiple at signs uses last separator",
			image:      "registry.example.com/team@name/image@sha256:abc123",
			wantName:   "registry.example.com/team@name/image",
			wantDigest: "sha256:abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotDigest := parseImageNameAndDigest(tt.image)
			if gotName != tt.wantName || gotDigest != tt.wantDigest {
				t.Fatalf("parseImageNameAndDigest(%q) = (%q, %q), want (%q, %q)",
					tt.image, gotName, gotDigest, tt.wantName, tt.wantDigest)
			}
		})
	}
}

func TestFormatDockerPin(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		wantParts   []string
	}{
		{
			name:      "without service",
			wantParts: []string{"[DOCKERFILE]", "ubuntu:latest", "ubuntu:latest@sha256:abc123"},
		},
		{
			name:        "with service",
			serviceName: "web",
			wantParts:   []string{"[COMPOSE]", "web", "nginx:alpine", "nginx:alpine@sha256:def456"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				if tt.serviceName == "" {
					FormatDockerPin("DOCKERFILE", "", "ubuntu:latest", "ubuntu:latest@sha256:abc123")
				} else {
					FormatDockerPin("COMPOSE", tt.serviceName, "nginx:alpine", "nginx:alpine@sha256:def456")
				}
			})

			for _, part := range tt.wantParts {
				if !strings.Contains(output, part) {
					t.Fatalf("FormatDockerPin output %q missing %q", output, part)
				}
			}
		})
	}
}

func TestFormatAlreadyPinnedMessage(t *testing.T) {
	tests := []struct {
		name      string
		fileType  string
		count     int
		wantParts []string
	}{
		{
			name:      "no docker references",
			fileType:  "DOCKERFILE",
			count:     0,
			wantParts: []string{"[DOCKERFILE]", "No dockerfile references found"},
		},
		{
			name:      "one image reference",
			fileType:  "COMPOSE",
			count:     1,
			wantParts: []string{"[COMPOSE]", "All 1 image already pinned"},
		},
		{
			name:      "multiple action references",
			fileType:  "ACTIONS",
			count:     2,
			wantParts: []string{"[ACTIONS]", "All 2 actions already pinned"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				FormatAlreadyPinnedMessage(tt.fileType, tt.count)
			})

			for _, part := range tt.wantParts {
				if !strings.Contains(output, part) {
					t.Fatalf("FormatAlreadyPinnedMessage output %q missing %q", output, part)
				}
			}
		})
	}
}

func TestFormatAlreadyPinnedActionsMessage(t *testing.T) {
	output := captureStdout(t, func() {
		FormatAlreadyPinnedActionsMessage([]string{
			"actions/checkout@0123456789012345678901234567890123456789",
			"invalid-action-reference",
		})
	})

	for _, part := range []string{"[ACTIONS]", "2 actions are already pinned", "actions/checkout@", "invalid-action-reference"} {
		if !strings.Contains(output, part) {
			t.Fatalf("FormatAlreadyPinnedActionsMessage output %q missing %q", output, part)
		}
	}

	emptyOutput := captureStdout(t, func() {
		FormatAlreadyPinnedActionsMessage(nil)
	})
	if !strings.Contains(emptyOutput, "No actions references found") {
		t.Fatalf("FormatAlreadyPinnedActionsMessage empty output = %q", emptyOutput)
	}
}

func TestFormatAlreadyPinnedDockerMessage(t *testing.T) {
	output := captureStdout(t, func() {
		FormatAlreadyPinnedDockerMessage("COMPOSE", []string{
			"nginx:alpine@sha256:abc123",
			"postgres:15@sha256:def456",
		}, []string{"web", ""})
	})

	for _, part := range []string{"[COMPOSE]", "2 images are already pinned", "web: nginx:alpine@sha256:abc123", "postgres:15@sha256:def456"} {
		if !strings.Contains(output, part) {
			t.Fatalf("FormatAlreadyPinnedDockerMessage output %q missing %q", output, part)
		}
	}

	emptyOutput := captureStdout(t, func() {
		FormatAlreadyPinnedDockerMessage("DOCKERFILE", nil, nil)
	})
	if !strings.Contains(emptyOutput, "No image references found") {
		t.Fatalf("FormatAlreadyPinnedDockerMessage empty output = %q", emptyOutput)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldNoColor := color.NoColor
	color.NoColor = true
	defer func() { color.NoColor = oldNoColor }()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}

	os.Stdout = writer
	fn()
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error: %v", err)
	}
	os.Stdout = oldStdout

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll() error: %v", err)
	}
	return string(output)
}
