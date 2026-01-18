//go:build simple

package main

// Simple build mode: no tree-sitter imports.
// This file exists to ensure the package compiles when built with -tags simple.
// In simple mode, only regex-based analysis is available.
