package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/goccy/go-yaml"
	"github.com/grantbirki/gh-pin/internal/version"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/types/ref"
)

var (
	noColor     = flag.Bool("no-color", false, "disable colored output")
	dryRun      = flag.Bool("dry-run", false, "preview changes without writing files")
	recursive   = flag.Bool("recursive", true, "scan directories recursively")
	showVersion = flag.Bool("version", false, "show version information")
	algo        = flag.String("algo", "sha256", "digest algorithm to check for (sha256, sha512, etc.)")
)

// ComposeFile is a minimal representation of a docker-compose YAML
type ComposeFile struct {
	Services map[string]struct {
		Image string `yaml:"image"`
	} `yaml:"services"`
}

// hasDigest checks if an image reference already contains a digest with the specified algorithm
func hasDigest(image, algorithm string) bool {
	return strings.Contains(image, "@"+algorithm+":")
}

// pinImage resolves an image tag to its immutable digest using regclient
func pinImage(rc *regclient.RegClient, image string) (string, error) {
	r, err := ref.New(image)
	if err != nil {
		return "", fmt.Errorf("parse ref %q: %w", image, err)
	}

	ctx := context.Background()
	m, err := rc.ManifestHead(ctx, r)
	if err != nil {
		return "", fmt.Errorf("fetch manifest for %q: %w", image, err)
	}

	digest := m.GetDescriptor().Digest
	return fmt.Sprintf("%s@%s", r.CommonName(), digest.String()), nil
}

// processCompose updates image tags in a Compose file
func processCompose(rc *regclient.RegClient, path string) error {
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
		if def.Image != "" && !hasDigest(def.Image, *algo) {
			pinned, err := pinImage(rc, def.Image)
			if err != nil {
				color.Yellow("WARN: %v", err)
				continue
			}
			if pinned != def.Image {
				color.Green("[COMPOSE] %s: %s → %s", svc, def.Image, pinned)
				cf.Services[svc] = struct {
					Image string `yaml:"image"`
				}{Image: pinned}
				changed = true
			}
		}
	}

	if changed && !*dryRun {
		out, err := yaml.Marshal(&cf)
		if err != nil {
			return err
		}
		return os.WriteFile(path, out, 0644)
	}

	return nil
}

// processDockerfile updates FROM lines in a Dockerfile
func processDockerfile(rc *regclient.RegClient, path string) error {
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
			if len(parts) >= 2 && !hasDigest(parts[1], *algo) {
				pinned, err := pinImage(rc, parts[1])
				if err != nil {
					color.Yellow("WARN: %v", err)
					output.WriteString(line + "\n")
					continue
				}
				color.Green("[DOCKERFILE] %s → %s", parts[1], pinned)
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

	if changed && !*dryRun {
		return os.WriteFile(path, output.Bytes(), 0644)
	}

	return nil
}

// scanPath walks a directory and processes all supported files
func scanPath(rc *regclient.RegClient, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if !*recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}

		lower := strings.ToLower(filepath.Base(path))
		ext := strings.ToLower(filepath.Ext(path))

		switch {
		case lower == "dockerfile" || strings.HasSuffix(lower, ".dockerfile"):
			fmt.Printf("Processing Dockerfile: %s\n", path)
			return processDockerfile(rc, path)
		case lower == "docker-compose.yml",
			lower == "docker-compose.yaml",
			(ext == ".yml" || ext == ".yaml"):
			// Try parsing as compose; skip if not valid
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var cf ComposeFile
			if err := yaml.Unmarshal(data, &cf); err != nil {
				return nil // skip non-compose YAMLs
			}
			if len(cf.Services) > 0 {
				fmt.Printf("Processing Compose: %s\n", path)
				return processCompose(rc, path)
			}
		}
		return nil
	})
}

func main() {
	flag.Parse()

	// Show version and exit if requested
	if *showVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}

	// Disable colors if requested
	if *noColor {
		color.NoColor = true
	}

	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [--version] [--dry-run] [--no-color] [--recursive=false] [--algo=sha256] <file|dir> [file|dir...]\n", os.Args[0])
		os.Exit(1)
	}

	rc := regclient.New()

	for _, target := range flag.Args() {
		info, err := os.Stat(target)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		if info.IsDir() {
			if err := scanPath(rc, target); err != nil {
				log.Fatalf("Error scanning %s: %v", target, err)
			}
		} else {
			lower := strings.ToLower(filepath.Base(target))
			ext := strings.ToLower(filepath.Ext(target))
			switch {
			case lower == "dockerfile" || strings.HasSuffix(lower, ".dockerfile"):
				if err := processDockerfile(rc, target); err != nil {
					log.Fatalf("Error processing %s: %v", target, err)
				}
			case ext == ".yml" || ext == ".yaml":
				if err := processCompose(rc, target); err != nil {
					log.Fatalf("Error processing %s: %v", target, err)
				}
			default:
				log.Printf("Skipping unsupported file: %s", target)
			}
		}
	}
}
