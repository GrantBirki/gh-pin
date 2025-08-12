package processor

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// FormatPinMessage formats a pin message with colored components
func FormatPinMessage(fileType, originalImage, pinnedImage string) {
	formatPinMessageWithService(fileType, "", originalImage, pinnedImage)
}

// FormatPinMessageWithService formats a pin message with colored components and service name
func FormatPinMessageWithService(fileType, serviceName, originalImage, pinnedImage string) {
	formatPinMessageWithService(fileType, serviceName, originalImage, pinnedImage)
}

// formatPinMessageWithService is the internal implementation
func formatPinMessageWithService(fileType, serviceName, originalImage, pinnedImage string) {
	// Parse the original image to separate name and tag
	imageName, imageTag := parseImageNameAndTag(originalImage)

	// Parse the pinned image to separate name and digest
	pinnedName, pinnedDigest := parseImageNameAndDigest(pinnedImage)

	if serviceName != "" {
		fmt.Printf("📌 [%s] %s: %s:%s → %s@%s\n",
			color.BlueString(fileType),
			color.MagentaString(serviceName),
			color.WhiteString(imageName),
			color.CyanString(imageTag),
			color.WhiteString(pinnedName),
			color.CyanString(pinnedDigest),
		)
	} else {
		fmt.Printf("📌 [%s] %s:%s → %s@%s\n",
			color.BlueString(fileType),
			color.WhiteString(imageName),
			color.CyanString(imageTag),
			color.WhiteString(pinnedName),
			color.CyanString(pinnedDigest),
		)
	}
}

// parseImageNameAndTag splits an image reference into name and tag
func parseImageNameAndTag(image string) (name, tag string) {
	// Find the last colon that's not part of a digest
	lastColon := -1
	for i := len(image) - 1; i >= 0; i-- {
		if image[i] == ':' {
			// Check if this colon is part of a digest (preceded by @)
			if i > 0 && image[i-1] == '@' {
				continue
			}
			// Check if there's an @ after this colon (digest format)
			hasAtAfter := false
			for j := i + 1; j < len(image); j++ {
				if image[j] == '@' {
					hasAtAfter = true
					break
				}
			}
			if !hasAtAfter {
				lastColon = i
				break
			}
		}
	}

	if lastColon == -1 {
		// No tag found, assume "latest"
		return image, "latest"
	}

	return image[:lastColon], image[lastColon+1:]
}

// parseImageNameAndDigest splits a pinned image reference into name and digest
func parseImageNameAndDigest(image string) (name, digest string) {
	// Find the @ symbol that separates name from digest
	atIndex := strings.LastIndex(image, "@")
	if atIndex == -1 {
		// No digest found, this shouldn't happen for pinned images
		return image, ""
	}

	return image[:atIndex], image[atIndex+1:]
}
