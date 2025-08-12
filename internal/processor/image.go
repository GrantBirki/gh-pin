package processor

import (
	"context"
	"fmt"
	"strings"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
)

// hasDigest checks if an image reference already contains a digest with the specified algorithm
func hasDigest(image, algorithm string) bool {
	return strings.Contains(image, "@"+algorithm+":")
}

// PinImage resolves an image tag to its immutable digest using regclient
func PinImage(rc *regclient.RegClient, image string, config ProcessorConfig) (string, error) {
	r, err := ref.New(image)
	if err != nil {
		return "", fmt.Errorf("parse ref %q: %w", image, err)
	}

	ctx := context.Background()
	m, err := rc.ManifestHead(ctx, r)
	if err != nil {
		return "", fmt.Errorf("fetch manifest for %q: %w", image, err)
	}

	digest := m.GetDescriptor().Digest

	// Use original reference format if ExpandRegistry is false, otherwise use CommonName
	var imageRef string
	if config.ExpandRegistry {
		imageRef = r.CommonName()
	} else {
		// Use the original reference, but we need to strip the tag if present
		// since we're adding the digest
		originalRef := r.Reference
		if idx := strings.LastIndex(originalRef, ":"); idx != -1 && !strings.Contains(originalRef[idx:], "@") {
			originalRef = originalRef[:idx]
		}
		imageRef = originalRef
	}

	return fmt.Sprintf("%s@%s", imageRef, digest.String()), nil
}
