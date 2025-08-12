package processor

import (
	"bufio"
	"bytes"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/regclient/regclient"
)

// ProcessDockerfile updates FROM lines in a Dockerfile
func ProcessDockerfile(rc *regclient.RegClient, path string, config ProcessorConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var output bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	changed := false

	for scanner.Scan() {
		line := scanner.Text()
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trim), "FROM ") {
			parts := strings.Fields(trim)
			if len(parts) >= 2 && !hasDigest(parts[1], config.Algorithm) {
				pinned, err := PinImage(rc, parts[1])
				if err != nil {
					color.Yellow("WARN: %v", err)
					output.WriteString(line + "\n")
					continue
				}
				color.Green("📌 [DOCKERFILE] %s → %s", parts[1], pinned)
				// Preserve indentation if any
				indent := strings.Repeat(" ", len(line)-len(strings.TrimLeft(line, " ")))
				output.WriteString(indent + "FROM " + pinned + "\n")
				changed = true
				continue
			}
		}
		output.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if changed && !config.DryRun {
		return os.WriteFile(path, output.Bytes(), getFileMode(path))
	}

	return nil
}
