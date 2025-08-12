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
	noColor     = flag.Bool("no-color", false, "disable colored output")
	dryRun      = flag.Bool("dry-run", false, "preview changes without writing files")
	recursive   = flag.Bool("recursive", true, "scan directories recursively")
	showVersion = flag.Bool("version", false, "show version information")
	algo        = flag.String("algo", "sha256", "digest algorithm to check for (sha256, sha512, etc.)")
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
		fmt.Fprintf(os.Stderr, "Usage: %s [--version] [--dry-run] [--no-color] [--recursive=false] [--algo=sha256] <file|dir> [file|dir...]\n", os.Args[0])
		os.Exit(1)
	}

	// Create processor configuration
	config := processor.ProcessorConfig{
		DryRun:    *dryRun,
		Algorithm: *algo,
		NoColor:   *noColor,
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
