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
func PinImage(rc *regclient.RegClient, image string) (string, error) {
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
	return fmt.Sprintf("%s@%s", r.CommonName(), digest.String()), nil
}
