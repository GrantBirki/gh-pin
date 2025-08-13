package processor

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/regclient/regclient"
)

// ProcessDockerfile updates FROM lines in a Dockerfile
func ProcessDockerfile(rc *regclient.RegClient, path string, config ProcessorConfig) error {
	return ProcessFileGeneric(path, config, func(data []byte, config ProcessorConfig) ([]byte, bool, error) {
		return processDockerfileContent(rc, data, config)
	})
}

// processDockerfileContent processes the content of a Dockerfile
func processDockerfileContent(rc *regclient.RegClient, data []byte, config ProcessorConfig) ([]byte, bool, error) {
	var output bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	changed := false
	var pinnedImages []string

	for scanner.Scan() {
		line := scanner.Text()
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trim), "FROM ") {
			parts := strings.Fields(trim)
			if len(parts) >= 2 && !hasDigest(parts[1], config.Algorithm) {
				pinned, err := PinImage(rc, parts[1], config)
				if err != nil {
					LogWarning("%v", err)
					output.WriteString(line + "\n")
					continue
				}
				if pinned != "" {
					FormatDockerPin("DOCKERFILE", "", parts[1], pinned)
					// Preserve indentation if any
					indent := strings.Repeat(" ", len(line)-len(strings.TrimLeft(line, " ")))
					output.WriteString(indent + "FROM " + pinned + "\n")
					changed = true
					continue
				}
			} else if len(parts) >= 2 && hasDigest(parts[1], config.Algorithm) {
				pinnedImages = append(pinnedImages, parts[1])
			}
		}
		output.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, false, err
	}

	// If no changes were made, provide detailed feedback about what's already pinned
	if !changed && len(pinnedImages) > 0 && !config.Quiet {
		FormatAlreadyPinnedDockerMessage("DOCKERFILE", pinnedImages, nil)
	}

	return output.Bytes(), changed, nil
}
