package processor

import (
	"os"

	"github.com/fatih/color"
	"github.com/goccy/go-yaml"
	"github.com/regclient/regclient"
)

// ProcessCompose updates image tags in a Compose file
func ProcessCompose(rc *regclient.RegClient, path string, config ProcessorConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cf ComposeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return err
	}

	changed := false
	for svc, def := range cf.Services {
		if def.Image != "" && !hasDigest(def.Image, config.Algorithm) {
			pinned, err := PinImage(rc, def.Image, config)
			if err != nil {
				color.Yellow("WARN: %v", err)
				continue
			}
			if pinned != def.Image {
				FormatPinMessageWithService("COMPOSE", svc, def.Image, pinned)
				cf.Services[svc] = struct {
					Image string `yaml:"image"`
				}{Image: pinned}
				changed = true
			}
		}
	}

	if changed && !config.DryRun {
		out, err := yaml.Marshal(&cf)
		if err != nil {
			return err
		}
		return os.WriteFile(path, out, getFileMode(path))
	}

	return nil
}
