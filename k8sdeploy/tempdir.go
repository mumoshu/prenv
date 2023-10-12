package k8sdeploy

import (
	"fmt"
	"os"
)

// CreateTempDir creates a temporary directory.
// It's the caller's responsibility to delete the directory.
func CreateTempDir() (*string, error) {
	f, err := os.CreateTemp("", tempDirPattern)
	if err != nil {
		return nil, err
	}

	dir := f.Name()

	if err := os.Remove(dir); err != nil {
		return nil, fmt.Errorf("unable to replace the temp file %s with a directory: %w", dir, err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &dir, nil
}
