package processor

import (
	"fmt"

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
	var cf ComposeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, false, err
	}

	changed := false
	var pinnedImages []string
	var serviceNames []string

	// Access services if they exist
	if servicesData, exists := cf["services"]; exists {
		if services, ok := servicesData.(map[string]interface{}); ok {
			for svcName, svcData := range services {
				if svcDef, ok := svcData.(map[string]interface{}); ok {
					if imageValue, exists := svcDef["image"]; exists {
						if imageStr, ok := imageValue.(string); ok && imageStr != "" {
							if !hasDigest(imageStr, config.Algorithm) {
								// Get pinned image - for compose files, we'll use the clean digest format
								pinned, err := PinImageWithComment(rc, imageStr, config, false)
								if err != nil {
									LogWarning("%v", err)
									continue
								}
								if pinned != "" {
									// Create a version with comment for display purposes only
									displayVersion := pinned
									if config.Platform == "" {
										// Only add comment for index digests, not platform-specific manifests
										imageName, imageTag := ParseImageNameAndTag(imageStr)
										if imageTag != "" {
											displayVersion += fmt.Sprintf(" # pin@%s:%s", imageName, imageTag)
										}
									}
									FormatDockerPin("COMPOSE", svcName, imageStr, displayVersion)
									// Update only the image field with clean version, preserving all other properties
									svcDef["image"] = pinned
									changed = true
								}
							} else {
								pinnedImages = append(pinnedImages, imageStr)
								serviceNames = append(serviceNames, svcName)
							}
						}
					}
				}
			}
		}
	}

	// If no changes were made, provide detailed feedback about what's already pinned
	if !changed && len(pinnedImages) > 0 && !config.Quiet {
		FormatAlreadyPinnedDockerMessage("COMPOSE", pinnedImages, serviceNames)
	}

	if changed {
		out, err := yaml.Marshal(cf)
		if err != nil {
			return nil, false, err
		}
		return out, true, nil
	}

	return data, false, nil
}
