package processor

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"

	"github.com/cli/go-gh/v2/pkg/api"
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
	return ProcessFileGeneric(path, config, func(data []byte, config ProcessorConfig) ([]byte, bool, error) {
		return processActionsContent(rc, data, config)
	})
}

// processActionsContent processes the content of a GitHub Actions workflow file
func processActionsContent(rc *regclient.RegClient, data []byte, config ProcessorConfig) ([]byte, bool, error) {
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
			pinRef := ExtractPinComment(suffix)
			if pinRef != "" {
				actionRef = updateActionRefWithPinComment(actionRef, pinRef)
			}

			// Parse the action reference
			ref, err := parseActionRef(actionRef)
			if err != nil {
				LogWarning("%v", err)
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
				LogWarning("failed to resolve %s@%s: %v", ref.Owner+"/"+ref.Repo, ref.Ref, err)
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
		return nil, false, err
	}

	return output.Bytes(), changed, nil
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
// DEPRECATED: Use ExtractPinComment from common.go instead
func extractPinComment(suffix string) string {
	return ExtractPinComment(suffix)
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
	return UpdateCommentWithPin(suffix, originalRef)
}

// resolveActionToSHA resolves a GitHub action tag/ref to a commit SHA using GitHub CLI's REST client
func resolveActionToSHA(ref *GitHubRef) (string, error) {
	// Create a REST client using the pre-hydrated GitHub CLI client
	restClient, err := api.DefaultRESTClient()
	if err != nil {
		return "", fmt.Errorf("failed to create GitHub REST client: %w", err)
	}

	// API endpoint path
	path := fmt.Sprintf("repos/%s/%s/commits/%s", ref.Owner, ref.Repo, ref.Ref)

	var commit struct {
		SHA string `json:"sha"`
	}

	// Make the API call using the GitHub CLI's REST client
	err = restClient.Get(path, &commit)
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s/%s@%s: %w", ref.Owner, ref.Repo, ref.Ref, err)
	}

	return commit.SHA, nil
}

// FormatActionPinMessage formats a pin message for GitHub Actions
func FormatActionPinMessage(originalRef, pinnedRef string) {
	// Parse original reference
	origParsed, _ := parseActionRef(originalRef)
	pinnedParsed, _ := parseActionRef(pinnedRef)

	if origParsed != nil && pinnedParsed != nil {
		originalDisplay := origParsed.Owner + "/" + origParsed.Repo + "@" + origParsed.Ref
		pinnedDisplay := pinnedParsed.Owner + "/" + pinnedParsed.Repo + "@" + pinnedParsed.Ref[:8] + "..."
		FormatReferenceMessage("ACTIONS", "", originalDisplay, pinnedDisplay)
	}
}
