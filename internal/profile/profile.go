// Package profile keeps hostu's profiles in profiles.yaml: the user's own
// lists of domains, settings by tdp E3, with the one written to the hosts
// file marked active.
package profile

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/vulcanshen/hostu/internal/hosts"
)

// File is the profiles' file in the settings directory.
const File = "profiles.yaml"

// Profile is a hosts file kept as hostu's own: a name and its domains.
type Profile struct {
	Name    string         `yaml:"name"`
	Domains []hosts.Domain `yaml:"domains"`
}

// Store is profiles.yaml.
type Store struct {
	// Active is the name of the profile last written to the hosts file.
	Active   string     `yaml:"active"`
	Profiles []*Profile `yaml:"profiles"`
	path     string
}

// Load reads the store in dir; found is false when it is not there yet.
func Load(dir string) (s *Store, found bool, err error) {
	s = &Store{path: filepath.Join(dir, File)}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, false, nil
	}
	if err != nil {
		return s, false, err
	}
	if err := yaml.Unmarshal(b, s); err != nil {
		return s, true, err
	}
	s.Profiles = slices.DeleteFunc(s.Profiles, func(p *Profile) bool { return p == nil })
	return s, true, nil
}

// Path is where the store is kept.
func (s *Store) Path() string { return s.path }

// Save writes the store: into a temporary file next to it first, then over
// it, so that a write cut short leaves the old one whole.
func (s *Store) Save() error {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(s); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), File+".*")
	if err != nil {
		return err
	}
	_, err = tmp.Write(b.Bytes())
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), s.path)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return err
}

// Get is the profile named name, nil for none.
func (s *Store) Get(name string) *Profile {
	i := slices.IndexFunc(s.Profiles, func(p *Profile) bool { return p.Name == name })
	if i < 0 {
		return nil
	}
	return s.Profiles[i]
}

// Taken reports whether another profile than but has name, in any case.
func (s *Store) Taken(name string, but *Profile) bool {
	return slices.ContainsFunc(s.Profiles, func(p *Profile) bool { return p != but && strings.EqualFold(p.Name, name) })
}

// Remove takes p out.
func (s *Store) Remove(p *Profile) {
	s.Profiles = slices.DeleteFunc(s.Profiles, func(o *Profile) bool { return o == p })
}
