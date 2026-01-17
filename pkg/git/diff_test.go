package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsGitRepo(t *testing.T) {
	// Test with a temp directory (not a git repo)
	tmpDir := t.TempDir()
	if IsGitRepo(tmpDir) {
		t.Error("IsGitRepo() returned true for non-git directory")
	}

	// If we're in a git repo, test that it returns true
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}

	// Check if the project is a git repo
	if IsGitRepo(wd) {
		t.Log("Current directory is a git repo")
	} else {
		t.Log("Current directory is not a git repo (skipping some tests)")
	}
}

func TestGetRepoRoot(t *testing.T) {
	// Test with non-git directory
	tmpDir := t.TempDir()
	_, err := GetRepoRoot(tmpDir)
	if err == nil {
		t.Error("GetRepoRoot() should return error for non-git directory")
	}
}

func TestRefExists(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}

	repoRoot, err := GetRepoRoot(wd)
	if err != nil {
		t.Skip("Not in a git repository")
	}

	// HEAD should always exist
	if !RefExists("HEAD", repoRoot) {
		t.Error("RefExists() returned false for HEAD")
	}

	// Random non-existent ref
	if RefExists("nonexistent-ref-12345", repoRoot) {
		t.Error("RefExists() returned true for non-existent ref")
	}
}

func TestGetChangedFiles(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}

	repoRoot, err := GetRepoRoot(wd)
	if err != nil {
		t.Skip("Not in a git repository")
	}

	// Get changed files from HEAD (should be empty or have staged changes)
	supportedExts := map[string]bool{".go": true}
	files, err := GetChangedFiles("HEAD", repoRoot, supportedExts)
	if err != nil {
		t.Fatalf("GetChangedFiles() error: %v", err)
	}

	// Just verify it returns without error and paths are absolute
	for _, f := range files {
		if !filepath.IsAbs(f) {
			t.Errorf("GetChangedFiles() returned relative path: %s", f)
		}
	}
	t.Logf("Found %d changed .go files since HEAD", len(files))
}

func TestGetChangedFilesFiltersByExtension(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}

	repoRoot, err := GetRepoRoot(wd)
	if err != nil {
		t.Skip("Not in a git repository")
	}

	// Only .xyz files (should return empty)
	supportedExts := map[string]bool{".xyz": true}
	files, err := GetChangedFiles("HEAD~100", repoRoot, supportedExts)
	if err != nil {
		// This might fail if there aren't 100 commits
		t.Skip("Not enough git history")
	}

	// Should have no .xyz files
	for _, f := range files {
		if filepath.Ext(f) != ".xyz" {
			t.Errorf("GetChangedFiles() returned file with wrong extension: %s", f)
		}
	}
}

func TestGetFileAtRef(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working directory: %v", err)
	}

	repoRoot, err := GetRepoRoot(wd)
	if err != nil {
		t.Skip("Not in a git repository")
	}

	// Try to get a known file at HEAD
	testFile := filepath.Join(repoRoot, "go.mod")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("go.mod not found in repo root")
	}

	content, err := GetFileAtRef("HEAD", testFile, repoRoot)
	if err != nil {
		t.Fatalf("GetFileAtRef() error: %v", err)
	}

	if len(content) == 0 {
		t.Error("GetFileAtRef() returned empty content for existing file")
	}

	// File that doesn't exist should return nil, nil
	nonExistent := filepath.Join(repoRoot, "nonexistent-file-12345.go")
	content, err = GetFileAtRef("HEAD", nonExistent, repoRoot)
	if err != nil {
		t.Errorf("GetFileAtRef() returned error for non-existent file: %v", err)
	}
	if content != nil {
		t.Error("GetFileAtRef() returned content for non-existent file")
	}
}
