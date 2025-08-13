package processor

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// FormatDockerPin formats a Docker image pin message with granular coloring
func FormatDockerPin(fileType, serviceName, originalImage, pinnedImage string) {
	// Parse the original image to separate name and tag
	imageName, imageTag := parseImageNameAndTag(originalImage)

	// Parse the pinned image to separate name and digest
	pinnedName, pinnedDigest := parseImageNameAndDigest(pinnedImage)

	if serviceName != "" {
		fmt.Printf("📌 [%s] %s: %s:%s → %s@%s\n",
			color.BlueString(fileType),
			color.CyanString(serviceName),
			color.WhiteString(imageName),
			color.GreenString(imageTag),
			color.WhiteString(pinnedName),
			color.GreenString(pinnedDigest),
		)
	} else {
		fmt.Printf("📌 [%s] %s:%s → %s@%s\n",
			color.BlueString(fileType),
			color.WhiteString(imageName),
			color.GreenString(imageTag),
			color.WhiteString(pinnedName),
			color.GreenString(pinnedDigest),
		)
	}
}

// FormatActionPin formats a GitHub Action pin message with granular coloring
func FormatActionPin(originalRef, pinnedRef string) {
	// Parse original reference
	origParsed, _ := parseActionRef(originalRef)
	pinnedParsed, _ := parseActionRef(pinnedRef)

	if origParsed != nil && pinnedParsed != nil {
		// Format: owner/repo@ref → owner/repo@sha
		fmt.Printf("📌 [%s] %s%s%s%s%s → %s%s%s%s%s\n",
			color.BlueString("ACTIONS"),
			color.WhiteString(origParsed.Owner),
			color.BlueString("/"),
			color.WhiteString(origParsed.Repo),
			color.BlueString("@"),
			color.GreenString(origParsed.Ref),
			color.WhiteString(pinnedParsed.Owner),
			color.BlueString("/"),
			color.WhiteString(pinnedParsed.Repo),
			color.BlueString("@"),
			color.GreenString(pinnedParsed.Ref),
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
