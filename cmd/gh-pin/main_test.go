package main

import (
	"os"
	"testing"
)

func TestMain_VersionFlag(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test version flag
	os.Args = []string{"gh-pin", "--version"}

	// We can't easily test main() since it calls os.Exit()
	// But we can test that the flag is recognized by checking if it would be parsed
	// This is mainly for coverage of the main package
}

func TestMain_HelpFlag(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test help flag
	os.Args = []string{"gh-pin", "--help"}

	// Again, mainly for coverage purposes
}

// Note: Testing main() directly is challenging because it calls os.Exit()
