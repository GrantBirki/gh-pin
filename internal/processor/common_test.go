package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessFileGeneric(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create test file
	err := os.WriteFile(testFile, []byte("original content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name           string
		config         ProcessorConfig
		processor      FileProcessor
		expectedOutput string
		expectChange   bool
		expectError    bool
	}{
		{
			name:   "no changes",
			config: ProcessorConfig{DryRun: false},
			processor: func(data []byte, config ProcessorConfig) ([]byte, bool, error) {
				return data, false, nil
			},
			expectedOutput: "original content",
			expectChange:   false,
		},
		{
			name:   "changes with dry run",
			config: ProcessorConfig{DryRun: true},
			processor: func(data []byte, config ProcessorConfig) ([]byte, bool, error) {
				return []byte("modified content"), true, nil
			},
			expectedOutput: "original content", // Should not change file in dry run
			expectChange:   true,
		},
		{
			name:   "changes without dry run",
			config: ProcessorConfig{DryRun: false},
			processor: func(data []byte, config ProcessorConfig) ([]byte, bool, error) {
				return []byte("modified content"), true, nil
			},
			expectedOutput: "modified content",
			expectChange:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset file content before each test
			err := os.WriteFile(testFile, []byte("original content"), 0644)
			if err != nil {
				t.Fatalf("Failed to reset test file: %v", err)
			}

			err = ProcessFileGeneric(testFile, tt.config, tt.processor)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check file content
			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}
			if string(content) != tt.expectedOutput {
				t.Errorf("Expected content %q, got %q", tt.expectedOutput, string(content))
			}
		})
	}
}

func TestUpdateCommentWithPin(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		originalRef string
		expected    string
	}{
		{
			name:        "empty text",
			text:        "",
			originalRef: "v4",
			expected:    " # pin@v4",
		},
		{
			name:        "text with existing comment",
			text:        " # some comment",
			originalRef: "v4",
			expected:    " # pin@v4 # some comment",
		},
		{
			name:        "text with existing pin comment",
			text:        " # pin@v5",
			originalRef: "v4",
			expected:    " # pin@v5", // Should not change
		},
		{
			name:        "text without comment",
			text:        " some suffix",
			originalRef: "v4",
			expected:    " some suffix # pin@v4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UpdateCommentWithPin(tt.text, tt.originalRef)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
