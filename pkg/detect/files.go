package detect

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zen-systems/hollowcheck/pkg/contract"
)

// DetectMissingFiles checks that all required files exist.
func DetectMissingFiles(baseDir string, files []contract.RequiredFile) (*DetectionResult, error) {
	result := &DetectionResult{}

	for _, f := range files {
		if !f.Required {
			continue
		}

		fullPath := filepath.Join(baseDir, f.Path)
		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				result.AddViolation(Violation{
					Rule:     RuleMissingFile,
					Message:  fmt.Sprintf("required file %q does not exist", f.Path),
					File:     f.Path,
					Line:     0,
					Severity: SeverityError,
				})
				continue
			}
			return nil, fmt.Errorf("checking file %s: %w", f.Path, err)
		}

		if info.IsDir() {
			result.AddViolation(Violation{
				Rule:     RuleMissingFile,
				Message:  fmt.Sprintf("required file %q is a directory, not a file", f.Path),
				File:     f.Path,
				Line:     0,
				Severity: SeverityError,
			})
		}
	}

	return result, nil
}
