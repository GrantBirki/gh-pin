package processor

import (
	"bufio"
	"bytes"
	"regexp"

	"github.com/goccy/go-yaml"
	"github.com/regclient/regclient"
)

// ProcessCompose updates image tags in a Compose file
func ProcessCompose(rc *regclient.RegClient, path string, config ProcessorConfig) error {
	return ProcessFileGeneric(path, config, func(data []byte, config ProcessorConfig) ([]byte, bool, error) {
		return processComposeContent(rc, data, config)
	})
}

// processComposeContent processes the content of a Docker Compose file
func processComposeContent(rc *regclient.RegClient, data []byte, config ProcessorConfig) ([]byte, bool, error) {
	// First, validate that the file is valid YAML by attempting to unmarshal it
	// This preserves the error handling behavior expected by tests
	var validationCheck interface{}
	if err := yaml.Unmarshal(data, &validationCheck); err != nil {
		return nil, false, err
	}

	var output bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	changed := false
	var pinnedImages []string
	var serviceNames []string

	// Regex pattern to match image lines in compose files
	// Matches: "image: nginx:latest" or "  image: nginx:latest" with optional comments
	imagePattern := regexp.MustCompile(`^(\s*image:\s*)([^\s#]+)(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()

		if match := imagePattern.FindStringSubmatch(line); match != nil {
			prefix := match[1]   // "  image: " including indentation
			imageRef := match[2] // "nginx:latest"
			suffix := match[3]   // " # some comment" or empty

			if !hasDigest(imageRef, config.Algorithm) {
				// Get pinned image using clean tag@digest format
				pinned, err := PinImage(rc, imageRef, config)
				if err != nil {
					LogWarning("%v", err)
					output.WriteString(line + "\n")
					continue
				}
				if pinned != "" {
					// Use clean format for both console display and file content
					FormatDockerPin("COMPOSE", "", imageRef, pinned)

					// Preserve the original line structure, only replacing the image reference
					newLine := prefix + pinned + suffix
					output.WriteString(newLine + "\n")
					changed = true
					continue
				}
			} else {
				pinnedImages = append(pinnedImages, imageRef)
				// Try to extract service name from context (simple heuristic)
				serviceNames = append(serviceNames, "")
			}
		}

		output.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, false, err
	}

	// If no changes were made, provide detailed feedback about what's already pinned
	if !changed && len(pinnedImages) > 0 && !config.Quiet {
		FormatAlreadyPinnedDockerMessage("COMPOSE", pinnedImages, serviceNames)
	}

	return output.Bytes(), changed, nil
}
