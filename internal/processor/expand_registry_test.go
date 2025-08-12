package processor

import (
	"fmt"
	"testing"

	"github.com/regclient/regclient/types/ref"
)

func TestExpandRegistryLogic(t *testing.T) {
	tests := []struct {
		name           string
		image          string
		expandRegistry bool
		expectedFormat string
	}{
		{
			name:           "preserve original format - ubuntu",
			image:          "ubuntu:latest",
			expandRegistry: false,
			expectedFormat: "ubuntu@<digest>",
		},
		{
			name:           "use expanded format - ubuntu",
			image:          "ubuntu:latest",
			expandRegistry: true,
			expectedFormat: "docker.io/library/ubuntu:latest@<digest>",
		},
		{
			name:           "preserve original format - nginx",
			image:          "nginx:alpine",
			expandRegistry: false,
			expectedFormat: "nginx@<digest>",
		},
		{
			name:           "use expanded format - nginx",
			image:          "nginx:alpine",
			expandRegistry: true,
			expectedFormat: "docker.io/library/nginx:alpine@<digest>",
		},
		{
			name:           "preserve original format - fully qualified registry",
			image:          "gcr.io/my-project/my-app:v1.0",
			expandRegistry: false,
			expectedFormat: "gcr.io/my-project/my-app@<digest>",
		},
		{
			name:           "use expanded format - fully qualified registry",
			image:          "gcr.io/my-project/my-app:v1.0",
			expandRegistry: true,
			expectedFormat: "gcr.io/my-project/my-app:v1.0@<digest>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := ref.New(tt.image)
			if err != nil {
				t.Fatalf("Failed to parse ref: %v", err)
			}

			// Test the logic that PinImage uses
			var imageRef string
			if tt.expandRegistry {
				imageRef = r.CommonName()
			} else {
				// Use the original reference, but strip the tag if present
				originalRef := r.Reference
				if idx := lastColonIndex(originalRef); idx != -1 && !containsAt(originalRef[idx:]) {
					originalRef = originalRef[:idx]
				}
				imageRef = originalRef
			}

			result := fmt.Sprintf("%s@<digest>", imageRef)

			if result != tt.expectedFormat {
				t.Errorf("Expected %s, got %s", tt.expectedFormat, result)
			}
		})
	}
}

// Helper functions (same as in our PinImage logic)
func lastColonIndex(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}

func containsAt(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '@' {
			return true
		}
	}
	return false
}
