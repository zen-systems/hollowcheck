package contract

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// ParseFile reads and parses a contract from a file path.
func ParseFile(path string) (*Contract, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening contract file: %w", err)
	}
	defer f.Close()

	return Parse(f)
}

// Parse reads and parses a contract from an io.Reader.
func Parse(r io.Reader) (*Contract, error) {
	var c Contract
	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)

	if err := decoder.Decode(&c); err != nil {
		return nil, fmt.Errorf("parsing contract YAML: %w", err)
	}

	return &c, nil
}

// ParseBytes parses a contract from a byte slice.
func ParseBytes(data []byte) (*Contract, error) {
	var c Contract
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing contract YAML: %w", err)
	}
	return &c, nil
}
