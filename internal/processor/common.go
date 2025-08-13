package processor

import (
	"os"
	"regexp"

	"github.com/fatih/color"
)

// FileProcessor represents a generic file processing function
type FileProcessor func(data []byte, config ProcessorConfig) ([]byte, bool, error)

// ProcessFileGeneric provides a common pattern for file processing operations
func ProcessFileGeneric(path string, config ProcessorConfig, processor FileProcessor) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	output, changed, err := processor(data, config)
	if err != nil {
		return err
	}

	if changed && !config.DryRun {
		return os.WriteFile(path, output, getFileMode(path))
	}

	return nil
}

// LogWarning provides consistent warning message formatting
func LogWarning(format string, args ...interface{}) {
	color.Yellow("WARN: "+format, args...)
}

// Common regex patterns used across processors
var (
	// PinCommentPattern matches pin directive comments like "# pin@v5"
	PinCommentPattern = regexp.MustCompile(`#\s*pin@([^\s]+)`)
)

// ExtractPinComment extracts pin directive from comment like "# pin@v5"
// This is now generic and can be used by any processor
func ExtractPinComment(text string) string {
	match := PinCommentPattern.FindStringSubmatch(text)
	if match != nil {
		return match[1]
	}
	return ""
}

// UpdateCommentWithPin adds or updates a pin comment in text
func UpdateCommentWithPin(text, originalRef string) string {
	// If there's already a pin comment, don't add another
	if ExtractPinComment(text) != "" {
		return text
	}

	// Add pin comment
	if text == "" {
		return " # pin@" + originalRef
	}

	// Insert pin comment before existing comment
	if commentStart := regexp.MustCompile(`#`).FindStringIndex(text); commentStart != nil {
		before := text[:commentStart[0]]
		after := text[commentStart[0]:]
		return before + "# pin@" + originalRef + " " + after
	}

	return text + " # pin@" + originalRef
}
