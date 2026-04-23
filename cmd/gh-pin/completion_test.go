package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandRunnerCompletionCompletesCurrentDirectoryPaths(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tempDir, "docs"), 0755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	for _, file := range []string{
		filepath.Join(tempDir, ".github", "workflows", "ci.yml"),
		filepath.Join(tempDir, ".github", "workflows", "release.yml"),
		filepath.Join(tempDir, "Dockerfile"),
		filepath.Join(tempDir, ".env"),
	} {
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
	}
	t.Chdir(tempDir)

	tests := []struct {
		name      string
		args      []string
		want      []string
		notWant   []string
		directive string
	}{
		{
			name:      "hidden directory after dot prefix",
			args:      []string{"__complete", ".g"},
			want:      []string{".github/"},
			directive: ":2",
		},
		{
			name:      "nested directory",
			args:      []string{"__complete", ".github/w"},
			want:      []string{".github/workflows/"},
			directive: ":2",
		},
		{
			name:      "nested workflow files",
			args:      []string{"__complete", ".github/workflows/"},
			want:      []string{".github/workflows/ci.yml", ".github/workflows/release.yml"},
			directive: ":0",
		},
		{
			name:      "empty prefix hides dotfiles like shell completion",
			args:      []string{"__complete", ""},
			want:      []string{"Dockerfile", "docs/"},
			notWant:   []string{".github/", ".env"},
			directive: ":0",
		},
		{
			name:      "missing path falls back to shell file completion",
			args:      []string{"__complete", "missing/path"},
			directive: ":0",
		},
	}

	runner := defaultCommandRunner()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := runner.run("gh-pin", tt.args, &stdout, &stderr)

			if code != 0 {
				t.Fatalf("run() exit code = %d, want 0; stderr=%q", code, stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
			got := strings.TrimSpace(stdout.String())
			for _, want := range tt.want {
				if !completionOutputContains(got, want) {
					t.Fatalf("completion output = %q, want %q", got, want)
				}
			}
			for _, notWant := range tt.notWant {
				if completionOutputContains(got, notWant) {
					t.Fatalf("completion output = %q, did not want %q", got, notWant)
				}
			}
			if !strings.HasSuffix(got, tt.directive) {
				t.Fatalf("completion output = %q, want directive %q", got, tt.directive)
			}
		})
	}
}

func TestCommandRunnerCompletionCompletesFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    []string
		notWant []string
	}{
		{
			name: "flag names with descriptions",
			args: []string{"__complete", "--d"},
			want: []string{"--dry-run\tpreview changes without writing files", ":4"},
		},
		{
			name:    "flag names without descriptions",
			args:    []string{"__completeNoDesc", "--d"},
			want:    []string{"--dry-run", ":4"},
			notWant: []string{"\tpreview changes without writing files"},
		},
		{
			name: "mode value",
			args: []string{"__complete", "--mode", "d"},
			want: []string{"docker\tpin container image references only", ":4"},
		},
		{
			name: "platform equals value",
			args: []string{"__complete", "--platform=linux/a"},
			want: []string{"linux/amd64\tLinux AMD64 manifest", "linux/arm64\tLinux ARM64 manifest", ":4"},
		},
		{
			name:    "recursive bool value only completes with equals",
			args:    []string{"__complete", "--recursive", ""},
			notWant: []string{"true\tscan nested directories", "false\tscan only the selected directory"},
		},
		{
			name: "recursive equals value",
			args: []string{"__complete", "--recursive=f"},
			want: []string{"false\tscan only the selected directory", ":4"},
		},
	}

	runner := defaultCommandRunner()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := runner.run("gh-pin", tt.args, &stdout, &stderr)

			if code != 0 {
				t.Fatalf("run() exit code = %d, want 0; stderr=%q", code, stderr.String())
			}
			got := strings.TrimSpace(stdout.String())
			for _, want := range tt.want {
				if !completionOutputContains(got, want) {
					t.Fatalf("completion output = %q, want %q", got, want)
				}
			}
			for _, notWant := range tt.notWant {
				if strings.Contains(got, notWant) {
					t.Fatalf("completion output = %q, did not want %q", got, notWant)
				}
			}
		})
	}
}

func completionOutputContains(output, want string) bool {
	for _, line := range strings.Split(output, "\n") {
		if line == want {
			return true
		}
	}
	return false
}
