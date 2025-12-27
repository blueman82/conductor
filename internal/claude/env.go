// Package claude provides utilities for invoking Claude CLI.
package claude

import (
	"os"
	"os/exec"
	"path/filepath"
)

// conductorTmpDir is the clean temp directory for Claude CLI invocations.
// Using a dedicated directory avoids VSCode socket files that crash Claude CLI
// when --settings flag is used (known bug: github.com/anthropics/claude-code/issues/7624).
var conductorTmpDir string

func init() {
	// Create conductor-specific temp directory
	conductorTmpDir = filepath.Join(os.TempDir(), "conductor-claude")
	os.MkdirAll(conductorTmpDir, 0755)
}

// SetCleanEnv configures a command to use a clean TMPDIR without VSCode sockets.
// This prevents Claude CLI crashes when using --settings flag.
func SetCleanEnv(cmd *exec.Cmd) {
	// Copy current environment
	cmd.Env = os.Environ()

	// Override TMPDIR to avoid VSCode socket files
	found := false
	for i, env := range cmd.Env {
		if len(env) > 7 && env[:7] == "TMPDIR=" {
			cmd.Env[i] = "TMPDIR=" + conductorTmpDir
			found = true
			break
		}
	}
	if !found {
		cmd.Env = append(cmd.Env, "TMPDIR="+conductorTmpDir)
	}
}

// GetCleanTmpDir returns the clean temp directory path for Claude CLI.
func GetCleanTmpDir() string {
	return conductorTmpDir
}
