// Package envfile loads optional dotenv files without overriding process environment variables.
package envfile

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/subosito/gotenv"
)

// Load loads path when it exists. Existing process environment variables take precedence.
func Load(path string) error {
	env, err := gotenv.Read(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	for key, value := range env {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
}

// LoadNearest searches the working directory and its parents for filename.
// It is intended for test commands that run from package subdirectories.
func LoadNearest(filename string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	for {
		path := filepath.Join(dir, filename)
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}
