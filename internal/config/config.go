package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Context is a named connection profile: API endpoint + organization.
type Context struct {
	APIURL       string `yaml:"api_url"`
	Organization string `yaml:"organization"`
}

// File is the on-disk config.yaml shape (kubectl-style contexts). It holds
// NO secrets — tokens live in separate 0600 files (see TokenPath).
type File struct {
	CurrentContext string             `yaml:"current-context"`
	Contexts       map[string]Context `yaml:"contexts"`
}

// Dir returns the syschecks config directory: $XDG_CONFIG_HOME/syschecks or
// ~/.config/syschecks.
func Dir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "syschecks")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "syschecks")
}

// Load reads config.yaml from dir. A missing file is not an error: it
// returns an empty File so first-run behaves sanely.
func Load(dir string) (*File, error) {
	b, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if os.IsNotExist(err) {
		return &File{Contexts: map[string]Context{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	if f.Contexts == nil {
		f.Contexts = map[string]Context{}
	}
	return &f, nil
}

// TokenPath returns the path to the token file for a given context name.
func TokenPath(dir, ctxName string) string {
	return filepath.Join(dir, "token-"+ctxName)
}

// WriteToken persists a token for ctxName, creating dir (0700) as needed and
// writing the token file with 0600 permissions since it is a secret.
func WriteToken(dir, ctxName, token string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(TokenPath(dir, ctxName), []byte(token), 0o600)
}

// ReadToken reads the token file for ctxName.
func ReadToken(dir, ctxName string) (string, error) {
	b, err := os.ReadFile(TokenPath(dir, ctxName))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
