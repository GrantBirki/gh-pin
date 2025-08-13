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

// FormatAlreadyPinnedMessage formats a message when all references are already pinned
func FormatAlreadyPinnedMessage(fileType string, count int) {
	if count == 0 {
		fmt.Printf("ℹ️  [%s] No %s references found\n",
			color.BlueString(fileType),
			strings.ToLower(fileType),
		)
	} else {
		referenceType := "image"
		if fileType == "ACTIONS" {
			referenceType = "action"
		}

		plural := ""
		if count != 1 {
			plural = "s"
		}

		fmt.Printf("ℹ️  [%s] All %d %s%s already pinned\n",
			color.BlueString(fileType),
			count,
			referenceType,
			plural,
		)
	}
}

// FormatAlreadyPinnedActionsMessage formats a detailed message showing already pinned actions
func FormatAlreadyPinnedActionsMessage(pinnedActions []string) {
	if len(pinnedActions) == 0 {
		fmt.Printf("ℹ️  [%s] No actions references found\n",
			color.BlueString("ACTIONS"),
		)
		return
	}

	// Determine singular/plural
	actionWord := "action"
	verbForm := "is"
	if len(pinnedActions) > 1 {
		actionWord = "actions"
		verbForm = "are"
	}

	fmt.Printf("ℹ️  [%s] %d %s %s already pinned:\n",
		color.BlueString("ACTIONS"),
		len(pinnedActions),
		actionWord,
		verbForm,
	)

	for _, action := range pinnedActions {
		ref, err := parseActionRef(action)
		if err == nil {
			fmt.Printf("   • %s%s%s%s%s\n",
				color.WhiteString(ref.Owner),
				color.BlueString("/"),
				color.WhiteString(ref.Repo),
				color.BlueString("@"),
				color.GreenString(ref.Ref),
			)
		} else {
			fmt.Printf("   • %s\n", color.WhiteString(action))
		}
	}
}

// FormatAlreadyPinnedDockerMessage formats a detailed message showing already pinned Docker images
func FormatAlreadyPinnedDockerMessage(fileType string, pinnedImages []string, serviceNames []string) {
	if len(pinnedImages) == 0 {
		fmt.Printf("ℹ️  [%s] No image references found\n",
			color.BlueString(fileType),
		)
		return
	}

	// Determine singular/plural
	imageWord := "image"
	verbForm := "is"
	if len(pinnedImages) > 1 {
		imageWord = "images"
		verbForm = "are"
	}

	fmt.Printf("ℹ️  [%s] %d %s %s already pinned:\n",
		color.BlueString(fileType),
		len(pinnedImages),
		imageWord,
		verbForm,
	)

	for i, image := range pinnedImages {
		imageName, imageDigest := parseImageNameAndDigest(image)

		if fileType == "COMPOSE" && i < len(serviceNames) && serviceNames[i] != "" {
			fmt.Printf("   • %s: %s%s%s\n",
				color.CyanString(serviceNames[i]),
				color.WhiteString(imageName),
				color.BlueString("@"),
				color.GreenString(imageDigest),
			)
		} else {
			fmt.Printf("   • %s%s%s\n",
				color.WhiteString(imageName),
				color.BlueString("@"),
				color.GreenString(imageDigest),
			)
		}
	}
}
