package clients

import (
	"fmt"
	"os"
)

// readFileIfExists reads a file if it exists, returning its contents.
func readFileIfExists(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path from provider config
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return data, nil
}
