//go:build !simple

package main

// Register tree-sitter language parsers.
// This import is only included in non-simple builds (builds without -tags simple).
import (
	_ "github.com/zen-systems/hollowcheck/pkg/parser/treesitter/languages"
)
