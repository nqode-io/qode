// Package env loads environment variables from .env files in the project tree.
package env

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

const dotEnvFile = ".env"

// Load reads a .env file from projectRoot and sets any variables that are
// not already present in the environment. If projectRoot is empty, the
// current working directory is used. A missing .env file is not an error.
func Load(projectRoot string) error {
	root := projectRoot
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("env: cannot determine working directory: %w", err)
		}
		root = wd
	}

	path := filepath.Join(root, dotEnvFile)
	vars, err := godotenv.Read(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("env: parsing %s: %w", path, err)
	}

	for key, value := range vars {
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("env: setting %s: %w", key, err)
			}
		}
	}
	return nil
}
