package processor

import (
	"encoding/json"
	"fmt"
	"os"
)

// StaticGitHubResolver resolves action refs from an in-memory mapping.
type StaticGitHubResolver map[string]string

// NewStaticGitHubResolverFromFile loads a JSON object mapping "owner/repo@ref" to a 40-character SHA.
func NewStaticGitHubResolverFromFile(path string) (StaticGitHubResolver, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var resolver StaticGitHubResolver
	if err := json.Unmarshal(data, &resolver); err != nil {
		return nil, err
	}
	if resolver == nil {
		return nil, fmt.Errorf("resolver map must be a JSON object")
	}

	return resolver, nil
}

// ResolveActionToSHA resolves a GitHub action tag/ref to a commit SHA from a static map.
func (r StaticGitHubResolver) ResolveActionToSHA(ref *GitHubRef) (string, error) {
	if ref == nil {
		return "", fmt.Errorf("nil GitHub action reference")
	}

	key := fmt.Sprintf("%s/%s@%s", ref.Owner, ref.Repo, ref.Ref)
	sha, ok := r[key]
	if !ok {
		return "", fmt.Errorf("no static GitHub action mapping for %s", key)
	}
	if !isSHA(sha) {
		return "", fmt.Errorf("static GitHub action mapping for %s is not a 40-character SHA", key)
	}

	return sha, nil
}
