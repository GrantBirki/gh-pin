package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func isCompletionRequest(args []string) bool {
	if len(args) == 0 {
		return false
	}
	return args[0] == "__complete" || args[0] == "__completeNoDesc"
}

type completionItem struct {
	value       string
	description string
}

const (
	shellCompDirectiveDefault    = 0
	shellCompDirectiveNoSpace    = 2
	shellCompDirectiveNoFileComp = 4
)

var flagCompletionItems = []completionItem{
	{value: "--no-color", description: "disable colored output"},
	{value: "--dry-run", description: "preview changes without writing files"},
	{value: "--recursive", description: "scan directories recursively"},
	{value: "--pervasive", description: "scan all YAML files"},
	{value: "--expand-registry", description: "expand short image names"},
	{value: "--version", description: "show version information"},
	{value: "--algo", description: "digest algorithm"},
	{value: "--mode", description: "force processing mode"},
	{value: "--quiet", description: "suppress no-change messages"},
	{value: "--platform", description: "platform-specific manifest digest"},
}

func (r commandRunner) complete(noDescriptions bool, args []string, stdout io.Writer) int {
	completions, directive := r.completionResults(args)
	for _, completion := range completions {
		if noDescriptions || completion.description == "" {
			fmt.Fprintln(stdout, completion.value)
			continue
		}
		fmt.Fprintf(stdout, "%s\t%s\n", completion.value, completion.description)
	}
	fmt.Fprintf(stdout, ":%d\n", directive)
	return 0
}

func (r commandRunner) completionResults(args []string) ([]completionItem, int) {
	current := ""
	if len(args) > 0 {
		current = args[len(args)-1]
	}

	if completions, ok := completeFlagValue(args); ok {
		return completions, shellCompDirectiveNoFileComp
	}

	if strings.HasPrefix(current, "-") {
		return completeFlags(current), shellCompDirectiveNoFileComp
	}

	return r.completePath(current)
}

func completeFlagValue(args []string) ([]completionItem, bool) {
	if len(args) == 0 {
		return nil, false
	}

	current := args[len(args)-1]
	if flagName, prefix, ok := strings.Cut(current, "="); ok {
		values, found := flagValueCompletions(flagName)
		if !found {
			return nil, false
		}
		return filterCompletionItems(values, prefix), true
	}

	if len(args) < 2 {
		return nil, false
	}
	values, found := separateFlagValueCompletions(args[len(args)-2])
	if !found {
		return nil, false
	}
	return filterCompletionItems(values, current), true
}

func separateFlagValueCompletions(flagName string) ([]completionItem, bool) {
	switch flagName {
	case "--algo", "--mode", "--platform":
		return flagValueCompletions(flagName)
	default:
		return nil, false
	}
}

func flagValueCompletions(flagName string) ([]completionItem, bool) {
	switch flagName {
	case "--recursive":
		return []completionItem{
			{value: "true", description: "scan nested directories"},
			{value: "false", description: "scan only the selected directory"},
		}, true
	case "--algo":
		return []completionItem{
			{value: "sha256", description: "standard OCI digest algorithm"},
			{value: "sha512", description: "alternate digest algorithm"},
		}, true
	case "--mode":
		return []completionItem{
			{value: "docker", description: "pin container image references only"},
			{value: "actions", description: "pin GitHub Actions references only"},
		}, true
	case "--platform":
		return []completionItem{
			{value: "linux/amd64", description: "Linux AMD64 manifest"},
			{value: "linux/arm64", description: "Linux ARM64 manifest"},
			{value: "linux/arm/v7", description: "Linux ARMv7 manifest"},
		}, true
	default:
		return nil, false
	}
}

func completeFlags(prefix string) []completionItem {
	return filterCompletionItems(flagCompletionItems, prefix)
}

func filterCompletionItems(items []completionItem, prefix string) []completionItem {
	var completions []completionItem
	for _, item := range items {
		if strings.HasPrefix(item.value, prefix) {
			completions = append(completions, item)
		}
	}
	return completions
}

func (r commandRunner) completePath(prefix string) ([]completionItem, int) {
	dir, namePrefix := filepath.Split(prefix)
	readDir := dir
	if readDir == "" {
		readDir = "."
	}

	entries, err := r.readDir(readDir)
	if err != nil {
		return nil, shellCompDirectiveDefault
	}

	includeHidden := strings.HasPrefix(namePrefix, ".")
	var completions []completionItem
	dirCount := 0
	for _, entry := range entries {
		name := entry.Name()
		if namePrefix == "" && strings.HasPrefix(name, ".") && !includeHidden {
			continue
		}
		if !strings.HasPrefix(name, namePrefix) {
			continue
		}

		value := dir + name
		if entry.IsDir() {
			value += string(os.PathSeparator)
			dirCount++
		}
		completions = append(completions, completionItem{value: value})
	}

	if len(completions) > 0 && dirCount == len(completions) {
		return completions, shellCompDirectiveNoSpace
	}
	return completions, shellCompDirectiveDefault
}
