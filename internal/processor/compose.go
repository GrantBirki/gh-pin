package processor

import (
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

	for svc, def := range cf.Services {
		if def.Image != "" {
			if !hasDigest(def.Image, config.Algorithm) {
				pinned, err := PinImage(rc, def.Image, config)
				if err != nil {
					LogWarning("%v", err)
					continue
				}
				if pinned != "" {
					FormatDockerPin("COMPOSE", svc, def.Image, pinned)
					cf.Services[svc] = struct {
						Image string `yaml:"image"`
					}{Image: pinned}
					changed = true
				}
			} else {
				pinnedImages = append(pinnedImages, def.Image)
				serviceNames = append(serviceNames, svc)
			}
		}
	}

	// If no changes were made, provide detailed feedback about what's already pinned
	if !changed && len(pinnedImages) > 0 && !config.Quiet {
		FormatAlreadyPinnedDockerMessage("COMPOSE", pinnedImages, serviceNames)
	}

	if changed {
		out, err := yaml.Marshal(&cf)
		if err != nil {
			return nil, false, err
		}
		return out, true, nil
	}

	return data, false, nil
}
