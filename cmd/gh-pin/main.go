package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
	"github.com/grantbirki/gh-pin/internal/processor"
	"github.com/grantbirki/gh-pin/internal/scanner"
	"github.com/grantbirki/gh-pin/internal/version"
	"github.com/regclient/regclient"
)

var (
	noColor        = flag.Bool("no-color", false, "disable colored output")
	dryRun         = flag.Bool("dry-run", false, "preview changes without writing files")
	recursive      = flag.Bool("recursive", true, "scan directories recursively")
	pervasive      = flag.Bool("pervasive", false, "scan all YAML files, not just docker-compose files")
	expandRegistry = flag.Bool("expand-registry", false, "expand short image names to fully qualified registry names")
	showVersion    = flag.Bool("version", false, "show version information")
	algo           = flag.String("algo", "sha256", "digest algorithm to check for (sha256, sha512, etc.)")
	forceMode      = flag.String("mode", "", "force processing mode: 'docker' for containers only, 'actions' for GitHub Actions only")
	quiet          = flag.Bool("quiet", false, "suppress informational messages when no changes are needed")
	platform       = flag.String("platform", "", "pin to platform-specific manifest digest (e.g., linux/amd64, linux/arm/v7)")
)

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
		fmt.Fprintf(os.Stderr, "Usage: %s [--version] [--dry-run] [--no-color] [--recursive=false] [--pervasive] [--expand-registry] [--algo=sha256] [--mode=docker|actions] [--quiet] [--platform=linux/amd64] <file|dir> [file|dir...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nSupported file types:\n")
		fmt.Fprintf(os.Stderr, "  - Dockerfiles (FROM statements)\n")
		fmt.Fprintf(os.Stderr, "  - Docker Compose files (image: fields)\n")
		fmt.Fprintf(os.Stderr, "  - GitHub Actions workflows (uses: statements)\n")
		fmt.Fprintf(os.Stderr, "  - Generic YAML files (with --pervasive flag)\n")
		fmt.Fprintf(os.Stderr, "\nPlatform-specific pinning:\n")
		fmt.Fprintf(os.Stderr, "  Use --platform=<arch> to pin to manifest-specific digests (e.g., linux/amd64, linux/arm/v7)\n")
		fmt.Fprintf(os.Stderr, "  Without --platform, images are pinned to index digests with human-readable comments\n")
		os.Exit(1)
	}

	// Create processor configuration
	config := processor.ProcessorConfig{
		DryRun:         *dryRun,
		Algorithm:      *algo,
		NoColor:        *noColor,
		Pervasive:      *pervasive,
		ExpandRegistry: *expandRegistry,
		ForceMode:      *forceMode,
		Quiet:          *quiet,
		Platform:       *platform,
		GitHubResolver: &processor.DefaultGitHubResolver{},
	}

	rc := regclient.New()

	for _, target := range flag.Args() {
		info, err := os.Stat(target)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		if info.IsDir() {
			if err := scanner.ScanPath(rc, target, config, *recursive); err != nil {
				log.Fatalf("Error scanning %s: %v", target, err)
			}
		} else {
			if err := scanner.ProcessSingleFile(rc, target, config); err != nil {
				log.Fatalf("Error processing %s: %v", target, err)
			}
		}
	}
}
