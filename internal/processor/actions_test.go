package processor

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regclient/regclient"
)

// MockGitHubResolver is a mock implementation of GitHubResolver for testing
type MockGitHubResolver struct {
	responses map[string]string // map of "owner/repo@ref" to SHA
}

// ResolveActionToSHA returns a mock SHA for testing
func (m *MockGitHubResolver) ResolveActionToSHA(ref *GitHubRef) (string, error) {
	key := ref.Owner + "/" + ref.Repo + "@" + ref.Ref
	if sha, exists := m.responses[key]; exists {
		return sha, nil
	}
	// Return a default mock SHA if not found in responses map
	return "abcdef1234567890123456789012345678901234", nil
}

// NewMockGitHubResolver creates a new mock resolver with default responses
func NewMockGitHubResolver() *MockGitHubResolver {
	return &MockGitHubResolver{
		responses: map[string]string{
			"actions/checkout@v4": "08eba0b27e820071cde6df949e0beb9ba4906955",
			"actions/checkout@v5": "08c6903cd8c0fde910a37f88322edcfb5dd907a8",
			"actions/setup-go@v4": "93397bea11091df50f3d7e59dc26a7711a8bcfbe",
			"actions/setup-go@v5": "d35c59abb061a4a6fb18e82ac0862c26744d6ab5",
			"actions/cache@v3":    "88522ab9f39a2ea568f7027eddc7d8d8bc9d59c8",
			"actions/cache@v4":    "0400d5f644dc74513175e3cd8d07132dd4860809",
		},
	}
}

func TestParseActionRef(t *testing.T) {
	tests := []struct {
		name      string
		actionRef string
		wantOwner string
		wantRepo  string
		wantRef   string
		wantErr   bool
	}{
		{
			name:      "valid action ref with tag",
			actionRef: "actions/checkout@v4",
			wantOwner: "actions",
			wantRepo:  "checkout",
			wantRef:   "v4",
			wantErr:   false,
		},
		{
			name:      "valid action ref with SHA",
			actionRef: "actions/setup-go@0aaccfd150d50ccaeb58ebd88d36e91967a5f35b",
			wantOwner: "actions",
			wantRepo:  "setup-go",
			wantRef:   "0aaccfd150d50ccaeb58ebd88d36e91967a5f35b",
			wantErr:   false,
		},
		{
			name:      "invalid format - no @",
			actionRef: "actions/checkout",
			wantErr:   true,
		},
		{
			name:      "invalid format - no slash",
			actionRef: "actions-checkout@v4",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseActionRef(tt.actionRef)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseActionRef() expected error, got nil")
				}
				if got != nil {
					t.Errorf("parseActionRef() expected nil result, got %v", got)
				}
			} else {
				if err != nil {
					t.Errorf("parseActionRef() unexpected error: %v", err)
				}
				if got == nil {
					t.Errorf("parseActionRef() got nil result")
					return
				}
				if got.Owner != tt.wantOwner {
					t.Errorf("parseActionRef() Owner = %v, want %v", got.Owner, tt.wantOwner)
				}
				if got.Repo != tt.wantRepo {
					t.Errorf("parseActionRef() Repo = %v, want %v", got.Repo, tt.wantRepo)
				}
				if got.Ref != tt.wantRef {
					t.Errorf("parseActionRef() Ref = %v, want %v", got.Ref, tt.wantRef)
				}
			}
		})
	}
}

func TestIsSHA(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want bool
	}{
		{
			name: "valid 40-char SHA",
			ref:  "0aaccfd150d50ccaeb58ebd88d36e91967a5f35b",
			want: true,
		},
		{
			name: "valid 40-char SHA uppercase",
			ref:  "0AACCFD150D50CCAEB58EBD88D36E91967A5F35B",
			want: true,
		},
		{
			name: "short SHA",
			ref:  "0aaccfd",
			want: false,
		},
		{
			name: "tag version",
			ref:  "v4",
			want: false,
		},
		{
			name: "tag with numbers",
			ref:  "v1.2.3",
			want: false,
		},
		{
			name: "invalid character",
			ref:  "0aaccfd150d50ccaeb58ebd88d36e91967a5f35g",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSHA(tt.ref)
			if got != tt.want {
				t.Errorf("isSHA() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractPinComment(t *testing.T) {
	tests := []struct {
		name   string
		suffix string
		want   string
	}{
		{
			name:   "pin comment v4",
			suffix: " # pin@v4",
			want:   "v4",
		},
		{
			name:   "pin comment with version",
			suffix: " # pin@v1.2.3",
			want:   "v1.2.3",
		},
		{
			name:   "pin comment with spaces",
			suffix: "   #   pin@v5  ",
			want:   "v5",
		},
		{
			name:   "no pin comment",
			suffix: " # some other comment",
			want:   "",
		},
		{
			name:   "empty suffix",
			suffix: "",
			want:   "",
		},
		{
			name:   "pin comment with additional text",
			suffix: " # pin@v4 some other comment",
			want:   "v4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPinComment(tt.suffix)
			if got != tt.want {
				t.Errorf("ExtractPinComment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateActionRefWithPinComment(t *testing.T) {
	tests := []struct {
		name      string
		actionRef string
		pinRef    string
		want      string
	}{
		{
			name:      "update tag with pin",
			actionRef: "actions/checkout@v3",
			pinRef:    "v4",
			want:      "actions/checkout@v4",
		},
		{
			name:      "update SHA with pin",
			actionRef: "actions/setup-go@abc123",
			pinRef:    "v5",
			want:      "actions/setup-go@v5",
		},
		{
			name:      "invalid action ref is unchanged",
			actionRef: "docker://alpine:3.18",
			pinRef:    "v5",
			want:      "docker://alpine:3.18",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateActionRefWithPinComment(tt.actionRef, tt.pinRef)
			if got != tt.want {
				t.Errorf("updateActionRefWithPinComment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateSuffixWithPinComment(t *testing.T) {
	tests := []struct {
		name        string
		suffix      string
		originalRef string
		want        string
	}{
		{
			name:        "empty suffix",
			suffix:      "",
			originalRef: "v4",
			want:        " # pin@v4",
		},
		{
			name:        "whitespace suffix",
			suffix:      "   ",
			originalRef: "v4",
			want:        "    # pin@v4",
		},
		{
			name:        "existing comment",
			suffix:      " # some comment",
			originalRef: "v4",
			want:        " # pin@v4 # some comment", // Preserves existing comment
		},
		{
			name:        "already has pin comment",
			suffix:      " # pin@v3",
			originalRef: "v4",
			want:        " # pin@v3", // Should not change
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updateSuffixWithPinComment(tt.suffix, tt.originalRef)
			if got != tt.want {
				t.Errorf("updateSuffixWithPinComment() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Note: We skip testing the actual HTTP resolveActionToSHA function in unit tests
// since it requires network access. This would be covered in integration tests.

func TestProcessActions(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		content string
		config  ProcessorConfig
		wantErr bool
	}{
		{
			name: "basic action pinning",
			content: `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5`,
			config: ProcessorConfig{DryRun: true, GitHubResolver: NewMockGitHubResolver()},
		},
		{
			name: "already pinned actions",
			content: `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@692973e3d937129bcbf40652eb9f2f61becf3332
      - uses: actions/setup-go@v5`,
			config: ProcessorConfig{DryRun: true, GitHubResolver: NewMockGitHubResolver()},
		},
		{
			name: "action with pin comment",
			content: `name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3 # pin@v4
      - uses: actions/setup-go@v5`,
			config: ProcessorConfig{DryRun: true, GitHubResolver: NewMockGitHubResolver()},
		},
		{
			name: "no actions workflow",
			content: `# This is not a workflow file
some: other
yaml: content`,
			config: ProcessorConfig{DryRun: true, GitHubResolver: NewMockGitHubResolver()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tempDir, "test.yml")
			err := os.WriteFile(testFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Create regclient (not used for actions but required by interface)
			rc := regclient.New()

			// Process the file
			err = ProcessActions(rc, testFile, tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ProcessActions() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ProcessActions() unexpected error: %v", err)
				}
			}

			// Read the file content after processing
			result, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read test file after processing: %v", err)
			}

			// For dry run, content should be unchanged
			if tt.config.DryRun {
				if string(result) != tt.content {
					t.Errorf("ProcessActions() in dry run mode changed file content")
				}
			}
		})
	}
}

func TestProcessActionsContent_RewritesAndPreservesComments(t *testing.T) {
	data := []byte(`name: test
jobs:
  test:
    steps:
      - uses: actions/checkout@v3 # existing comment
      uses: actions/setup-go@v5
`)
	config := ProcessorConfig{DryRun: false, GitHubResolver: NewMockGitHubResolver()}

	output, changed, err := processActionsContent(data, config)
	if err != nil {
		t.Fatalf("processActionsContent() error: %v", err)
	}
	if !changed {
		t.Fatal("processActionsContent() changed = false, want true")
	}

	got := string(output)
	for _, want := range []string{
		"      - uses: actions/checkout@abcdef1234567890123456789012345678901234 # pin@v3 # existing comment",
		"      uses: actions/setup-go@d35c59abb061a4a6fb18e82ac0862c26744d6ab5 # pin@v5",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output:\n%s\nmissing %q", got, want)
		}
	}
}

func TestProcessActionsContent_LeavesInvalidReferencesUnchanged(t *testing.T) {
	data := []byte("steps:\n  - uses: docker://alpine:3.18\n")
	output, changed, err := processActionsContent(data, ProcessorConfig{
		DryRun:         false,
		GitHubResolver: NewMockGitHubResolver(),
	})
	if err != nil {
		t.Fatalf("processActionsContent() error: %v", err)
	}
	if changed {
		t.Fatal("processActionsContent() changed = true, want false")
	}
	if string(output) != string(data) {
		t.Fatalf("output = %q, want original %q", string(output), string(data))
	}
}

func TestProcessActionsContent_ResolverErrorLeavesLineUnchanged(t *testing.T) {
	data := []byte("steps:\n  - uses: actions/checkout@v4\n")
	output, changed, err := processActionsContent(data, ProcessorConfig{
		DryRun:         false,
		GitHubResolver: errorResolver{},
	})
	if err != nil {
		t.Fatalf("processActionsContent() error: %v", err)
	}
	if changed {
		t.Fatal("processActionsContent() changed = true, want false")
	}
	if string(output) != string(data) {
		t.Fatalf("output = %q, want original %q", string(output), string(data))
	}
}

func TestProcessActions_WritesPinnedFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "workflow.yml")
	content := "steps:\n  - uses: actions/checkout@v4\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("os.WriteFile() error: %v", err)
	}

	err := ProcessActions(nil, testFile, ProcessorConfig{
		DryRun:         false,
		GitHubResolver: NewMockGitHubResolver(),
	})
	if err != nil {
		t.Fatalf("ProcessActions() error: %v", err)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("os.ReadFile() error: %v", err)
	}
	want := "steps:\n  - uses: actions/checkout@08eba0b27e820071cde6df949e0beb9ba4906955 # pin@v4\n"
	if string(got) != want {
		t.Fatalf("file content = %q, want %q", string(got), want)
	}
}

func TestIsAlreadyPinnedToSHA(t *testing.T) {
	tests := []struct {
		name      string
		actionRef string
		want      bool
	}{
		{
			name:      "pinned to SHA",
			actionRef: "actions/checkout@692973e3d937129bcbf40652eb9f2f61becf3332",
			want:      true,
		},
		{
			name:      "tag reference",
			actionRef: "actions/checkout@v4",
			want:      false,
		},
		{
			name:      "invalid format",
			actionRef: "invalid-ref",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAlreadyPinnedToSHA(tt.actionRef)
			if got != tt.want {
				t.Errorf("isAlreadyPinnedToSHA() = %v, want %v", got, tt.want)
			}
		})
	}
}

type errorResolver struct{}

func (errorResolver) ResolveActionToSHA(*GitHubRef) (string, error) {
	return "", errors.New("resolver failed")
}
