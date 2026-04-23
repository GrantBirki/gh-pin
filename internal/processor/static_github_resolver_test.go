package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStaticGitHubResolverResolveActionToSHA(t *testing.T) {
	resolver := StaticGitHubResolver{
		"actions/checkout@v4": "08eba0b27e820071cde6df949e0beb9ba4906955",
	}

	sha, err := resolver.ResolveActionToSHA(&GitHubRef{Owner: "actions", Repo: "checkout", Ref: "v4"})
	if err != nil {
		t.Fatalf("ResolveActionToSHA() error: %v", err)
	}
	if sha != "08eba0b27e820071cde6df949e0beb9ba4906955" {
		t.Fatalf("sha = %q, want static mapping", sha)
	}
}

func TestStaticGitHubResolverErrors(t *testing.T) {
	tests := []struct {
		name     string
		resolver StaticGitHubResolver
		ref      *GitHubRef
	}{
		{
			name:     "nil ref",
			resolver: StaticGitHubResolver{},
			ref:      nil,
		},
		{
			name:     "missing mapping",
			resolver: StaticGitHubResolver{},
			ref:      &GitHubRef{Owner: "actions", Repo: "checkout", Ref: "v4"},
		},
		{
			name: "invalid sha",
			resolver: StaticGitHubResolver{
				"actions/checkout@v4": "not-a-sha",
			},
			ref: &GitHubRef{Owner: "actions", Repo: "checkout", Ref: "v4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.resolver.ResolveActionToSHA(tt.ref)
			if err == nil {
				t.Fatal("ResolveActionToSHA() error = nil, want error")
			}
		})
	}
}

func TestNewStaticGitHubResolverFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "actions.json")
	if err := os.WriteFile(path, []byte(`{"actions/checkout@v4":"08eba0b27e820071cde6df949e0beb9ba4906955"}`), 0644); err != nil {
		t.Fatalf("write resolver file: %v", err)
	}

	resolver, err := NewStaticGitHubResolverFromFile(path)
	if err != nil {
		t.Fatalf("NewStaticGitHubResolverFromFile() error: %v", err)
	}

	sha, err := resolver.ResolveActionToSHA(&GitHubRef{Owner: "actions", Repo: "checkout", Ref: "v4"})
	if err != nil {
		t.Fatalf("ResolveActionToSHA() error: %v", err)
	}
	if sha != "08eba0b27e820071cde6df949e0beb9ba4906955" {
		t.Fatalf("sha = %q, want static mapping", sha)
	}
}

func TestNewStaticGitHubResolverFromFileErrors(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		path    string
		content []byte
		write   bool
	}{
		{
			name:  "missing file",
			path:  filepath.Join(tempDir, "missing.json"),
			write: false,
		},
		{
			name:    "invalid json",
			path:    filepath.Join(tempDir, "invalid.json"),
			content: []byte(`{`),
			write:   true,
		},
		{
			name:    "not object",
			path:    filepath.Join(tempDir, "array.json"),
			content: []byte(`[]`),
			write:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.write {
				if err := os.WriteFile(tt.path, tt.content, 0644); err != nil {
					t.Fatalf("write resolver file: %v", err)
				}
			}

			_, err := NewStaticGitHubResolverFromFile(tt.path)
			if err == nil {
				t.Fatal("NewStaticGitHubResolverFromFile() error = nil, want error")
			}
		})
	}
}
