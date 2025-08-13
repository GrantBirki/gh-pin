package processor

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/platform"
	"github.com/regclient/regclient/types/ref"
)

// hasDigest checks if an image reference already contains a digest with the specified algorithm
func hasDigest(image, algorithm string) bool {
	return strings.Contains(image, "@"+algorithm+":")
}

// PinImage resolves an image tag to its immutable digest using regclient
func PinImage(rc *regclient.RegClient, image string, config ProcessorConfig) (string, error) {
	return PinImageWithComment(rc, image, config, true)
}

// PinImageWithComment resolves an image tag to its immutable digest using regclient
// includeComment controls whether to add human-readable comments for index digests
func PinImageWithComment(rc *regclient.RegClient, image string, config ProcessorConfig, includeComment bool) (string, error) {
	r, err := ref.New(image)
	if err != nil {
		return "", fmt.Errorf("parse ref %q: %w", image, err)
	}

	ctx := context.Background()

	// Store the original tag for human-readable comment
	originalTag := r.Tag

	var digest string
	var usePlatformSpecific bool

	// If platform is specified, try to get platform-specific manifest digest
	if config.Platform != "" {
		platformDigest, err := getPlatformSpecificDigest(ctx, rc, r, config.Platform)
		if err != nil {
			// Log warning and fall back to index digest
			log.Printf("Warning: Could not find manifest for platform %s: %v. Falling back to index digest.", config.Platform, err)
		} else {
			digest = platformDigest
			usePlatformSpecific = true
		}
	}

	// If no platform specified or platform-specific lookup failed, get index digest
	if digest == "" {
		m, err := rc.ManifestHead(ctx, r)
		if err != nil {
			return "", fmt.Errorf("fetch manifest for %q: %w", image, err)
		}
		digest = m.GetDescriptor().Digest.String()
	}

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

	// Format the result based on whether we're using platform-specific or index digest
	if usePlatformSpecific && originalTag != "" {
		// For platform-specific manifests, use the full tag@digest format
		return fmt.Sprintf("%s:%s@%s", imageRef, originalTag, digest), nil
	} else {
		// For index digests, use image@digest format with optional human-readable comment
		result := fmt.Sprintf("%s@%s", imageRef, digest)
		if originalTag != "" && includeComment {
			result += fmt.Sprintf(" # pin@%s:%s", imageRef, originalTag)
		}
		return result, nil
	}
}

// getPlatformSpecificDigest retrieves the digest for a specific platform manifest
func getPlatformSpecificDigest(ctx context.Context, rc *regclient.RegClient, r ref.Ref, platformStr string) (string, error) {
	// Parse the platform string
	plat, err := platform.Parse(platformStr)
	if err != nil {
		return "", fmt.Errorf("invalid platform %q: %w", platformStr, err)
	}

	// Get the manifest list/index
	m, err := rc.ManifestGet(ctx, r)
	if err != nil {
		return "", fmt.Errorf("fetch manifest for %q: %w", r.Reference, err)
	}

	// Check if it's a manifest list/index
	if !m.IsList() {
		return "", fmt.Errorf("image %q is not a multi-platform image", r.Reference)
	}

	// Find the platform-specific manifest
	desc, err := manifest.GetPlatformDesc(m, &plat)
	if err != nil {
		return "", fmt.Errorf("platform %q not found in manifest list: %w", platformStr, err)
	}

	return desc.Digest.String(), nil
}
