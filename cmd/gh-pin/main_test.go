package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/grantbirki/gh-pin/internal/processor"
	"github.com/regclient/regclient"
)

type fakeFileInfo struct {
	name  string
	isDir bool
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() fs.FileMode  { return 0644 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.isDir }
func (f fakeFileInfo) Sys() any           { return nil }

func TestRunVersionFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"--version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run() exit code = %d, want 0", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("built at")) {
		t.Fatalf("version output %q does not contain build metadata", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestCommandRunnerNoArgsPrintsUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := defaultTestRunner(nil).run("gh-pin", nil, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("run() exit code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Usage: gh-pin")) {
		t.Fatalf("stderr = %q, want usage", stderr.String())
	}
}

func TestCommandRunnerFlagParseError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := defaultTestRunner(nil).run("gh-pin", []string{"--definitely-not-a-flag"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("run() exit code = %d, want 2", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("flag provided but not defined")) {
		t.Fatalf("stderr = %q, want flag parse error", stderr.String())
	}
}

func TestCommandRunnerRoutesFileAndDirectoryTargets(t *testing.T) {
	var stdout, stderr bytes.Buffer
	var scanCalls []struct {
		target    string
		config    processor.ProcessorConfig
		recursive bool
	}
	var processCalls []struct {
		target string
		config processor.ProcessorConfig
	}

	runner := defaultTestRunner(map[string]fakeFileInfo{
		"repo":       {name: "repo", isDir: true},
		"Dockerfile": {name: "Dockerfile", isDir: false},
	})
	runner.scanPath = func(_ *regclient.RegClient, target string, config processor.ProcessorConfig, recursive bool) error {
		scanCalls = append(scanCalls, struct {
			target    string
			config    processor.ProcessorConfig
			recursive bool
		}{target: target, config: config, recursive: recursive})
		return nil
	}
	runner.processSingleFile = func(_ *regclient.RegClient, target string, config processor.ProcessorConfig) error {
		processCalls = append(processCalls, struct {
			target string
			config processor.ProcessorConfig
		}{target: target, config: config})
		return nil
	}

	oldNoColor := color.NoColor
	defer func() { color.NoColor = oldNoColor }()

	args := []string{
		"--dry-run",
		"--recursive=false",
		"--pervasive",
		"--expand-registry",
		"--algo=sha512",
		"--mode=docker",
		"--quiet",
		"--platform=linux/amd64",
		"--no-color",
		"repo",
		"Dockerfile",
	}

	code := runner.run("gh-pin", args, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run() exit code = %d, want 0; stderr=%q", code, stderr.String())
	}
	if len(scanCalls) != 1 || scanCalls[0].target != "repo" || scanCalls[0].recursive {
		t.Fatalf("scan calls = %#v, want one non-recursive repo scan", scanCalls)
	}
	if len(processCalls) != 1 || processCalls[0].target != "Dockerfile" {
		t.Fatalf("process calls = %#v, want one Dockerfile process call", processCalls)
	}

	config := scanCalls[0].config
	if !config.DryRun || config.Algorithm != "sha512" || !config.NoColor ||
		!config.Pervasive || !config.ExpandRegistry || config.ForceMode != "docker" ||
		!config.Quiet || config.Platform != "linux/amd64" || config.GitHubResolver == nil {
		t.Fatalf("config = %#v, want flags reflected in processor config", config)
	}
	if !color.NoColor {
		t.Fatal("color.NoColor = false, want true after --no-color")
	}
}

func TestCommandRunnerStatError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runner := defaultTestRunner(nil)

	code := runner.run("gh-pin", []string{"missing"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("run() exit code = %d, want 1", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Error:")) {
		t.Fatalf("stderr = %q, want error prefix", stderr.String())
	}
}

func TestCommandRunnerProcessingErrors(t *testing.T) {
	tests := []struct {
		name    string
		targets map[string]fakeFileInfo
		setup   func(*commandRunner)
		args    []string
		wantErr []byte
	}{
		{
			name:    "scan error",
			targets: map[string]fakeFileInfo{"repo": {name: "repo", isDir: true}},
			setup: func(r *commandRunner) {
				r.scanPath = func(_ *regclient.RegClient, _ string, _ processor.ProcessorConfig, _ bool) error {
					return errors.New("scan failed")
				}
			},
			args:    []string{"repo"},
			wantErr: []byte("Error scanning repo: scan failed"),
		},
		{
			name:    "single file error",
			targets: map[string]fakeFileInfo{"Dockerfile": {name: "Dockerfile", isDir: false}},
			setup: func(r *commandRunner) {
				r.processSingleFile = func(_ *regclient.RegClient, _ string, _ processor.ProcessorConfig) error {
					return errors.New("process failed")
				}
			},
			args:    []string{"Dockerfile"},
			wantErr: []byte("Error processing Dockerfile: process failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			runner := defaultTestRunner(tt.targets)
			tt.setup(&runner)

			code := runner.run("gh-pin", tt.args, &stdout, &stderr)

			if code != 1 {
				t.Fatalf("run() exit code = %d, want 1", code)
			}
			if !bytes.Contains(stderr.Bytes(), tt.wantErr) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), string(tt.wantErr))
			}
		})
	}
}

func TestCommandRunnerResolverLoadError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runner := defaultTestRunner(map[string]fakeFileInfo{
		"Dockerfile": {name: "Dockerfile", isDir: false},
	})
	runner.loadGitHubResolverEnv = func(processor.GitHubResolver) (processor.GitHubResolver, error) {
		return nil, errors.New("bad resolver map")
	}

	code := runner.run("gh-pin", []string{"Dockerfile"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("run() exit code = %d, want 1", code)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("bad resolver map")) {
		t.Fatalf("stderr = %q, want resolver load error", stderr.String())
	}
}

func TestLoadGitHubResolverFromEnv(t *testing.T) {
	fallback := &processor.DefaultGitHubResolver{}
	t.Setenv("GH_PIN_GITHUB_RESOLVER_MAP", "")

	got, err := loadGitHubResolverFromEnv(fallback)
	if err != nil {
		t.Fatalf("loadGitHubResolverFromEnv() error: %v", err)
	}
	if got != fallback {
		t.Fatalf("loadGitHubResolverFromEnv() = %#v, want fallback", got)
	}

	mapPath := filepath.Join(t.TempDir(), "actions.json")
	if err := os.WriteFile(mapPath, []byte(`{"actions/checkout@v4":"08eba0b27e820071cde6df949e0beb9ba4906955"}`), 0644); err != nil {
		t.Fatalf("write resolver map: %v", err)
	}
	t.Setenv("GH_PIN_GITHUB_RESOLVER_MAP", mapPath)

	got, err = loadGitHubResolverFromEnv(fallback)
	if err != nil {
		t.Fatalf("loadGitHubResolverFromEnv() with map error: %v", err)
	}
	sha, err := got.ResolveActionToSHA(&processor.GitHubRef{Owner: "actions", Repo: "checkout", Ref: "v4"})
	if err != nil {
		t.Fatalf("ResolveActionToSHA() error: %v", err)
	}
	if sha != "08eba0b27e820071cde6df949e0beb9ba4906955" {
		t.Fatalf("sha = %q, want static mapping", sha)
	}
}

func TestLoadGitHubResolverFromEnvErrors(t *testing.T) {
	t.Setenv("GH_PIN_GITHUB_RESOLVER_MAP", filepath.Join(t.TempDir(), "missing.json"))

	_, err := loadGitHubResolverFromEnv(&processor.DefaultGitHubResolver{})
	if err == nil {
		t.Fatal("loadGitHubResolverFromEnv() error = nil, want missing file error")
	}
}

func TestSplitEnvList(t *testing.T) {
	got := splitEnvList(" registry.local:5000, ,127.0.0.1:5001 ")
	want := []string{"registry.local:5000", "127.0.0.1:5001"}
	if len(got) != len(want) {
		t.Fatalf("splitEnvList() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitEnvList()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func defaultTestRunner(files map[string]fakeFileInfo) commandRunner {
	return commandRunner{
		stat: func(path string) (os.FileInfo, error) {
			info, ok := files[path]
			if !ok {
				return nil, os.ErrNotExist
			}
			return info, nil
		},
		readDir: os.ReadDir,
		newRegClient: func() *regclient.RegClient {
			return nil
		},
		scanPath: func(*regclient.RegClient, string, processor.ProcessorConfig, bool) error {
			return nil
		},
		processSingleFile: func(*regclient.RegClient, string, processor.ProcessorConfig) error {
			return nil
		},
		gitHubResolver: &processor.DefaultGitHubResolver{},
		loadGitHubResolverEnv: func(fallback processor.GitHubResolver) (processor.GitHubResolver, error) {
			return fallback, nil
		},
	}
}
