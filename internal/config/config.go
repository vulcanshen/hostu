// Package config reads config.yaml, the user's settings (tdp E3).
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// File is the settings file in the settings directory.
const File = "config.yaml"

// Config is the settings hostu has: the family's input_tab_width only.
type Config struct {
	// InputTabWidth is how many spaces a tab in the text hostu shows is drawn
	// as (tdp E3); 4 when unset.
	InputTabWidth int `yaml:"input_tab_width"`
}

// Default is the settings with nothing set.
var Default = Config{InputTabWidth: 4}

// Path is config.yaml in dir.
func Path(dir string) string { return filepath.Join(dir, File) }

// Load reads config.yaml in dir: no file is the defaults, and so is a
// setting left out. A file that cannot be read gives the defaults and the
// error.
func Load(dir string) (Config, error) {
	c := Default
	b, err := os.ReadFile(Path(dir))
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err == nil {
		err = yaml.Unmarshal(b, &c)
	}
	if err == nil && c.InputTabWidth < 1 {
		err = fmt.Errorf("the setting input_tab_width is %d: it is at least 1", c.InputTabWidth)
	}
	if err != nil {
		return Default, err
	}
	return c, nil
}

// WriteDefault writes config.yaml in dir with the defaults: the new one in
// place of one that could not be read (tdp E3).
func WriteDefault(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(Path(dir), fmt.Appendf(nil, "input_tab_width: %d\n", Default.InputTabWidth), 0o644)
}
