package processor

import (
	"context"
	"strings"
	"testing"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
)

func TestHasDigest(t *testing.T) {
	tests := []struct {
		name      string
		image     string
		algorithm string
		expected  bool
	}{
		{
			name:      "image with sha256 digest",
			image:     "nginx@sha256:abc123",
			algorithm: "sha256",
			expected:  true,
		},
		{
			name:      "image with sha512 digest",
			image:     "nginx@sha512:def456",
			algorithm: "sha512",
			expected:  true,
		},
		{
			name:      "image without digest",
			image:     "nginx:latest",
			algorithm: "sha256",
			expected:  false,
		},
		{
			name:      "image with different algorithm digest",
			image:     "nginx@sha256:abc123",
			algorithm: "sha512",
			expected:  false,
		},
		{
			name:      "image with tag and digest",
			image:     "nginx:latest@sha256:abc123",
			algorithm: "sha256",
			expected:  true,
		},
		{
			name:      "empty image",
			image:     "",
			algorithm: "sha256",
			expected:  false,
		},
		{
			name:      "empty algorithm",
			image:     "nginx@sha256:abc123",
			algorithm: "",
			expected:  false,
		},
		{
			name:      "malformed digest",
			image:     "nginx@abc123",
			algorithm: "sha256",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasDigest(tt.image, tt.algorithm)
			if result != tt.expected {
				t.Errorf("hasDigest(%q, %q) = %v, expected %v",
					tt.image, tt.algorithm, result, tt.expected)
			}
		})
	}
}

func TestPinImage_InvalidReference(t *testing.T) {
	_, err := PinImage(regclient.New(), "not a valid image ref!", ProcessorConfig{Algorithm: "sha256"})
	if err == nil {
		t.Fatal("PinImage() error = nil, want parse error")
	}
	if !strings.Contains(err.Error(), "parse ref") {
		t.Fatalf("PinImage() error = %q, want parse ref context", err.Error())
	}
}

func TestGetPlatformSpecificDigest_InvalidPlatform(t *testing.T) {
	imageRef, err := ref.New("nginx:latest")
	if err != nil {
		t.Fatalf("ref.New() error: %v", err)
	}

	_, err = getPlatformSpecificDigest(context.Background(), nil, imageRef, "linux/amd64!")
	if err == nil {
		t.Fatal("getPlatformSpecificDigest() error = nil, want invalid platform error")
	}
	if !strings.Contains(err.Error(), "invalid platform") {
		t.Fatalf("getPlatformSpecificDigest() error = %q, want invalid platform context", err.Error())
	}
}
