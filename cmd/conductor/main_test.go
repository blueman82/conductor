package main

import (
	"testing"
)

func TestVersionConstant(t *testing.T) {
	if Version == "" {
		t.Error("Version constant should not be empty")
	}
}

func TestVersionExists(t *testing.T) {
	// This test verifies that Version constant exists and is accessible
	_ = Version
}
