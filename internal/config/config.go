// Package config reads config.yaml, the user's settings (tdp E3).
package config

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

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

// Misfit is a setting hostu can't use: its key, what it was, and the
// default used in its place (tdp E3, a value that does not fit).
type Misfit struct{ Key, Was, Now string }

// Path is config.yaml in dir.
func Path(dir string) string { return filepath.Join(dir, File) }

// Load reads config.yaml in dir: no file is the defaults, and so is a
// setting left out. A value hostu can't use is a misfit, its default used;
// a key hostu doesn't read is in notRead, and stays in the file (tdp E3). A
// file that cannot be read gives the defaults and the error.
func Load(dir string) (c Config, misfits []Misfit, notRead []string, err error) {
	c = Default
	doc, err := read(dir)
	if err != nil || doc == nil {
		return Default, nil, nil, err
	}
	m := doc.Content[0]
	for i := 0; i+1 < len(m.Content); i += 2 {
		key, v := m.Content[i].Value, m.Content[i+1]
		if key != "input_tab_width" {
			notRead = append(notRead, key)
			continue
		}
		var n int
		if v.Decode(&n) != nil || n < 1 {
			misfits = append(misfits, Misfit{Key: key, Was: was(v), Now: strconv.Itoa(Default.InputTabWidth)})
			continue
		}
		c.InputTabWidth = n
	}
	return c, misfits, notRead, nil
}

// read is config.yaml in dir as yaml has it: nil when there is none or it
// sets nothing, an error when it is not a mapping of settings.
func read(dir string) (*yaml.Node, error) {
	b, err := os.ReadFile(Path(dir))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Tag == "!!null" {
		return nil, nil
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("not a mapping of settings")
	}
	return &doc, nil
}

// was is a value as the file has it, in words when it is not one.
func was(v *yaml.Node) string {
	switch v.Kind {
	case yaml.SequenceNode:
		return "a list"
	case yaml.MappingNode:
		return "a mapping"
	}
	return v.Value
}

// Fix writes config.yaml in dir again with each misfit's default in its
// place, everything else as it was, the keys hostu doesn't read too: into a
// temporary file next to it first, then over it.
func Fix(dir string, misfits []Misfit) error {
	doc, err := read(dir)
	if err != nil || doc == nil {
		return err
	}
	m := doc.Content[0]
	for i := 0; i+1 < len(m.Content); i += 2 {
		for _, f := range misfits {
			if m.Content[i].Value == f.Key {
				m.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: f.Now}
			}
		}
	}
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return err
	}
	path := Path(dir)
	tmp, err := os.CreateTemp(dir, File+".*")
	if err != nil {
		return err
	}
	_, err = tmp.Write(b.Bytes())
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		if info, serr := os.Stat(path); serr == nil {
			err = os.Chmod(tmp.Name(), info.Mode().Perm())
		}
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return err
}
