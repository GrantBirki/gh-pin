package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/grantbirki/gh-pin/internal/processor"
	"github.com/grantbirki/gh-pin/internal/scanner"
	"github.com/grantbirki/gh-pin/internal/version"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
)

type commandRunner struct {
	stat                  func(string) (os.FileInfo, error)
	readDir               func(string) ([]os.DirEntry, error)
	newRegClient          func() *regclient.RegClient
	scanPath              func(*regclient.RegClient, string, processor.ProcessorConfig, bool) error
	processSingleFile     func(*regclient.RegClient, string, processor.ProcessorConfig) error
	gitHubResolver        processor.GitHubResolver
	loadGitHubResolverEnv func(processor.GitHubResolver) (processor.GitHubResolver, error)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return defaultCommandRunner().run(os.Args[0], args, stdout, stderr)
}

func defaultCommandRunner() commandRunner {
	return commandRunner{
		stat:                  os.Stat,
		readDir:               os.ReadDir,
		newRegClient:          newRegClientFromEnv,
		scanPath:              scanner.ScanPath,
		processSingleFile:     scanner.ProcessSingleFile,
		gitHubResolver:        &processor.DefaultGitHubResolver{},
		loadGitHubResolverEnv: loadGitHubResolverFromEnv,
	}
}

func (r commandRunner) run(programName string, args []string, stdout, stderr io.Writer) int {
	if isCompletionRequest(args) {
		return r.complete(args[0] == "__completeNoDesc", args[1:], stdout)
	}

	flags := flag.NewFlagSet(programName, flag.ContinueOnError)
	flags.SetOutput(stderr)

	noColor := flags.Bool("no-color", false, "disable colored output")
	dryRun := flags.Bool("dry-run", false, "preview changes without writing files")
	recursive := flags.Bool("recursive", true, "scan directories recursively")
	pervasive := flags.Bool("pervasive", false, "scan all YAML files, not just docker-compose files")
	expandRegistry := flags.Bool("expand-registry", false, "expand short image names to fully qualified registry names")
	showVersion := flags.Bool("version", false, "show version information")
	algo := flags.String("algo", "sha256", "digest algorithm to check for (sha256, sha512, etc.)")
	forceMode := flags.String("mode", "", "force processing mode: 'docker' for containers only, 'actions' for GitHub Actions only")
	quiet := flags.Bool("quiet", false, "suppress informational messages when no changes are needed")
	platform := flags.String("platform", "", "pin to platform-specific manifest digest (e.g., linux/amd64, linux/arm/v7)")

	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, version.String())
		return 0
	}

	if *noColor {
		color.NoColor = true
	}

	targets := flags.Args()
	if len(targets) == 0 {
		printUsage(stderr, programName)
		return 1
	}

	gitHubResolver := r.gitHubResolver
	if r.loadGitHubResolverEnv != nil {
		var err error
		gitHubResolver, err = r.loadGitHubResolverEnv(gitHubResolver)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
	}

	config := processor.ProcessorConfig{
		DryRun:         *dryRun,
		Algorithm:      *algo,
		NoColor:        *noColor,
		Pervasive:      *pervasive,
		ExpandRegistry: *expandRegistry,
		ForceMode:      *forceMode,
		Quiet:          *quiet,
		Platform:       *platform,
		GitHubResolver: gitHubResolver,
	}

	rc := r.newRegClient()

	for _, target := range targets {
		info, err := r.stat(target)
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		if info.IsDir() {
			if err := r.scanPath(rc, target, config, *recursive); err != nil {
				fmt.Fprintf(stderr, "Error scanning %s: %v\n", target, err)
				return 1
			}
		} else {
			if err := r.processSingleFile(rc, target, config); err != nil {
				fmt.Fprintf(stderr, "Error processing %s: %v\n", target, err)
				return 1
			}
		}
	}

	return 0
}

func newRegClientFromEnv() *regclient.RegClient {
	var opts []regclient.Opt

	for _, registry := range splitEnvList(os.Getenv("GH_PIN_INSECURE_REGISTRIES")) {
		host := *config.HostNewName(registry)
		host.TLS = config.TLSDisabled
		host.Hostname = registry
		opts = append(opts, regclient.WithConfigHost(host))
	}

	return regclient.New(opts...)
}

func loadGitHubResolverFromEnv(fallback processor.GitHubResolver) (processor.GitHubResolver, error) {
	path := strings.TrimSpace(os.Getenv("GH_PIN_GITHUB_RESOLVER_MAP"))
	if path == "" {
		return fallback, nil
	}

	resolver, err := processor.NewStaticGitHubResolverFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load static GitHub resolver map: %w", err)
	}
	return resolver, nil
}

func splitEnvList(value string) []string {
	var items []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func printUsage(stderr io.Writer, programName string) {
	fmt.Fprintf(stderr, "Usage: %s [--version] [--dry-run] [--no-color] [--recursive=false] [--pervasive] [--expand-registry] [--algo=sha256] [--mode=docker|actions] [--quiet] [--platform=linux/amd64] <file|dir> [file|dir...]\n", programName)
	fmt.Fprintf(stderr, "\nSupported file types:\n")
	fmt.Fprintf(stderr, "  - Dockerfiles (FROM statements)\n")
	fmt.Fprintf(stderr, "  - Docker Compose files (image: fields)\n")
	fmt.Fprintf(stderr, "  - GitHub Actions workflows (uses: statements)\n")
	fmt.Fprintf(stderr, "  - Generic YAML files (with --pervasive flag)\n")
	fmt.Fprintf(stderr, "\nPlatform-specific pinning:\n")
	fmt.Fprintf(stderr, "  Use --platform=<arch> to pin to manifest-specific digests (e.g., linux/amd64, linux/arm/v7)\n")
	fmt.Fprintf(stderr, "  Without --platform, images are pinned to index digests with human-readable comments\n")
}
