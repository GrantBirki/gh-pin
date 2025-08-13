package processor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/regclient/regclient"
)

// GitHubRef represents a GitHub repository reference
type GitHubRef struct {
	Owner string
	Repo  string
	Ref   string
	SHA   string
}

// usesPattern matches GitHub Actions 'uses:' statements
var usesPattern = regexp.MustCompile(`^(\s*)(uses:\s*)([^\s#]+)(.*)$`)

// actionRefPattern parses action references like "owner/repo@ref"
var actionRefPattern = regexp.MustCompile(`^([^/]+)/([^@]+)@(.+)$`)

// ProcessActions updates GitHub Actions workflow files to pin action references to commit SHAs
func ProcessActions(rc *regclient.RegClient, path string, config ProcessorConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var output bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	changed := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this line contains a uses: statement
		if match := usesPattern.FindStringSubmatch(line); match != nil {
			indent := match[1]
			usesPrefix := match[2]
			actionRef := match[3]
			suffix := match[4] // includes any comments

			// Check if action is already pinned to a SHA (40-char hex)
			if isAlreadyPinnedToSHA(actionRef) {
				output.WriteString(line + "\n")
				continue
			}

			// Check for pin comment directive
			pinRef := extractPinComment(suffix)
			if pinRef != "" {
				actionRef = updateActionRefWithPinComment(actionRef, pinRef)
			}

			// Parse the action reference
			ref, err := parseActionRef(actionRef)
			if err != nil {
				color.Yellow("WARN: %v", err)
				output.WriteString(line + "\n")
				continue
			}

			// Skip if already pinned to SHA
			if isSHA(ref.Ref) {
				output.WriteString(line + "\n")
				continue
			}

			// Resolve the tag/ref to a commit SHA
			sha, err := resolveActionToSHA(ref)
			if err != nil {
				color.Yellow("WARN: failed to resolve %s@%s: %v", ref.Owner+"/"+ref.Repo, ref.Ref, err)
				output.WriteString(line + "\n")
				continue
			}

			// Create the pinned reference
			pinnedRef := fmt.Sprintf("%s/%s@%s", ref.Owner, ref.Repo, sha)

			// Format the pin message
			FormatActionPinMessage(actionRef, pinnedRef)

			// Update the line with pinned reference, preserving indentation and comments
			newSuffix := updateSuffixWithPinComment(suffix, ref.Ref)
			newLine := indent + usesPrefix + pinnedRef + newSuffix
			output.WriteString(newLine + "\n")
			changed = true
			continue
		}

		output.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if changed && !config.DryRun {
		return os.WriteFile(path, output.Bytes(), getFileMode(path))
	}

	return nil
}

// parseActionRef parses an action reference like "owner/repo@ref"
func parseActionRef(actionRef string) (*GitHubRef, error) {
	match := actionRefPattern.FindStringSubmatch(actionRef)
	if match == nil {
		return nil, fmt.Errorf("invalid action reference format: %s", actionRef)
	}

	return &GitHubRef{
		Owner: match[1],
		Repo:  match[2],
		Ref:   match[3],
	}, nil
}

// isAlreadyPinnedToSHA checks if an action is already pinned to a commit SHA
func isAlreadyPinnedToSHA(actionRef string) bool {
	ref, err := parseActionRef(actionRef)
	if err != nil {
		return false
	}
	return isSHA(ref.Ref)
}

// isSHA checks if a string looks like a git commit SHA (40 character hex)
func isSHA(ref string) bool {
	if len(ref) != 40 {
		return false
	}
	for _, c := range ref {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// extractPinComment extracts pin directive from comment like "# pin@v5"
func extractPinComment(suffix string) string {
	// Look for pattern like "# pin@v5" or "# pin@v1.2.3"
	pinPattern := regexp.MustCompile(`#\s*pin@([^\s]+)`)
	match := pinPattern.FindStringSubmatch(suffix)
	if match != nil {
		return match[1]
	}
	return ""
}

// updateActionRefWithPinComment updates action ref based on pin comment
func updateActionRefWithPinComment(actionRef, pinRef string) string {
	ref, err := parseActionRef(actionRef)
	if err != nil {
		return actionRef
	}
	return fmt.Sprintf("%s/%s@%s", ref.Owner, ref.Repo, pinRef)
}

// updateSuffixWithPinComment adds or updates the pin comment in the suffix
func updateSuffixWithPinComment(suffix, originalRef string) string {
	// If there's already a pin comment, don't add another
	if extractPinComment(suffix) != "" {
		return suffix
	}

	// Add pin comment
	if strings.TrimSpace(suffix) == "" {
		// For empty or whitespace-only suffix, append pin comment
		return suffix + " # pin@" + originalRef
	}

	// Insert pin comment before existing comment
	if strings.Contains(suffix, "#") {
		commentStart := strings.Index(suffix, "#")
		before := suffix[:commentStart]
		after := suffix[commentStart:]
		return before + "# pin@" + originalRef + " " + after
	}

	return suffix + " # pin@" + originalRef
}

// resolveActionToSHA resolves a GitHub action tag/ref to a commit SHA using GitHub API
func resolveActionToSHA(ref *GitHubRef) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", ref.Owner, ref.Repo, ref.Ref)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return "", err
	}

	// Add User-Agent header as required by GitHub API
	req.Header.Set("User-Agent", "gh-pin")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d for %s/%s@%s", resp.StatusCode, ref.Owner, ref.Repo, ref.Ref)
	}

	var commit struct {
		SHA string `json:"sha"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return "", err
	}

	return commit.SHA, nil
}

// FormatActionPinMessage formats a pin message for GitHub Actions
func FormatActionPinMessage(originalRef, pinnedRef string) {
	// Parse original reference
	origParsed, _ := parseActionRef(originalRef)
	pinnedParsed, _ := parseActionRef(pinnedRef)

	if origParsed != nil && pinnedParsed != nil {
		fmt.Printf("📌 [%s] %s@%s → %s@%s\n",
			color.BlueString("ACTIONS"),
			color.WhiteString(origParsed.Owner+"/"+origParsed.Repo),
			color.CyanString(origParsed.Ref),
			color.WhiteString(pinnedParsed.Owner+"/"+pinnedParsed.Repo),
			color.CyanString(pinnedParsed.Ref[:8]+"..."), // Show abbreviated SHA
		)
	}
}
