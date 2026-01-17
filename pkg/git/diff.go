// Package git provides git integration for hollowcheck.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetChangedFiles returns files changed since the given git ref.
// Only files with supported extensions are returned.
// Paths are returned as absolute paths.
func GetChangedFiles(ref string, basePath string, supportedExts map[string]bool) ([]string, error) {
	// Get the absolute path for the base
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Run git diff to get changed files
	cmd := exec.Command("git", "diff", "--name-only", ref, "--", basePath)
	cmd.Dir = absBase

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff failed: %w, stderr: %s", err, stderr.String())
	}

	// Parse output
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	var files []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if file has supported extension
		ext := filepath.Ext(line)
		if supportedExts != nil && !supportedExts[ext] {
			continue
		}

		// Convert to absolute path
		absPath := filepath.Join(absBase, line)
		files = append(files, absPath)
	}

	return files, nil
}

// GetFileAtRef returns the contents of a file at a specific git ref.
func GetFileAtRef(ref string, filePath string, repoRoot string) ([]byte, error) {
	// Convert to relative path from repo root
	relPath, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative path: %w", err)
	}

	// Run git show
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", ref, relPath))
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// File might not exist at that ref - that's OK
		if strings.Contains(stderr.String(), "does not exist") ||
			strings.Contains(stderr.String(), "exists on disk, but not in") {
			return nil, nil
		}
		return nil, fmt.Errorf("git show failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// GetRepoRoot returns the root directory of the git repository.
func GetRepoRoot(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = absPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("not a git repository: %w, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// IsGitRepo checks if the given path is inside a git repository.
func IsGitRepo(path string) bool {
	_, err := GetRepoRoot(path)
	return err == nil
}

// RefExists checks if a git ref exists.
func RefExists(ref string, repoRoot string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	cmd.Dir = repoRoot
	return cmd.Run() == nil
}
