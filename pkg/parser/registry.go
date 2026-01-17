package parser

import "sync"

var (
	registry = make(map[string]func() Parser)
	mu       sync.RWMutex
)

// Register adds a parser factory for a file extension.
// Extension should include the dot (e.g., ".go", ".py").
func Register(ext string, factory func() Parser) {
	mu.Lock()
	defer mu.Unlock()
	registry[ext] = factory
}

// ForExtension returns a parser for the given file extension.
// Returns nil, false if no parser is registered for the extension.
func ForExtension(ext string) (Parser, bool) {
	mu.RLock()
	defer mu.RUnlock()
	factory, ok := registry[ext]
	if !ok {
		return nil, false
	}
	return factory(), true
}

// SupportedExtensions returns all registered file extensions.
func SupportedExtensions() []string {
	mu.RLock()
	defer mu.RUnlock()
	exts := make([]string, 0, len(registry))
	for ext := range registry {
		exts = append(exts, ext)
	}
	return exts
}
