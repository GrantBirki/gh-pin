package processor

import (
	"testing"
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

// Note: PinImage() requires regclient and network access,
// so we'll test it in integration tests or with mocking if needed.
// For now, we focus on the testable logic without external dependencies.
