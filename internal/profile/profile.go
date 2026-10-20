// Package profile keeps hostu's profiles in profiles.yaml: the user's own
// lists of domains, data by tdp E3, with the one written to the hosts file
// marked active.
package profile

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/vulcanshen/hostu/internal/hosts"
)

// File is the profiles' file in the data directory.
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

// Load reads the store in dir; found is false when it is not there yet. A
// file that can't be read, or that check finds wrong, gives a store with no
// profiles and the error: nothing of it is shown or written.
func Load(dir string) (s *Store, found bool, err error) {
	path := filepath.Join(dir, File)
	s = &Store{path: path}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, false, nil
	}
	if err != nil {
		return s, false, err
	}
	if err := yaml.Unmarshal(b, s); err != nil {
		return &Store{path: path}, true, err
	}
	s.Profiles = slices.DeleteFunc(s.Profiles, func(p *Profile) bool { return p == nil })
	if err := s.check(); err != nil {
		return &Store{path: path}, true, err
	}
	return s, true, nil
}

// check holds a hand-edited file to what hostu itself writes, so that no
// name it can't show and no line it shouldn't write gets in: names as
// CheckName has them, each once; domains and addresses a hosts file takes.
func (s *Store) check() error {
	for i, p := range s.Profiles {
		if err := CheckName(p.Name); err != nil {
			why := err.Error() // a sentence for the input it is made for
			return fmt.Errorf("profile %d is named %q: %s", i+1, p.Name, strings.ToLower(why[:1])+why[1:])
		}
		if s.Taken(p.Name, p) {
			return fmt.Errorf("two profiles are named %s", p.Name)
		}
		for j, d := range p.Domains {
			if err := hosts.CheckDomain(d.Name); err != nil {
				return fmt.Errorf("the profile %s has %q, which is not a domain", p.Name, d.Name)
			}
			if hosts.Has(p.Domains[:j], d.Name) {
				return fmt.Errorf("the profile %s has %s twice", p.Name, d.Name)
			}
			if len(d.Addresses) == 0 {
				return fmt.Errorf("the profile %s has %s with no address", p.Name, d.Name)
			}
			for _, a := range d.Addresses {
				if err := hosts.CheckAddress(a); err != nil {
					return fmt.Errorf("the profile %s has %q for %s, which is not an IPv4 or IPv6 address", p.Name, a, d.Name)
				}
			}
		}
	}
	if s.Active != "" && s.Get(s.Active) == nil {
		return fmt.Errorf("the active profile %q is not there", s.Active)
	}
	return nil
}

// CheckName says what is wrong with s as a profile's name: no space at
// either end, no control character. Empty is the caller's to say.
func CheckName(s string) error {
	if s != strings.TrimSpace(s) {
		return errors.New("A name has no space at either end")
	}
	if strings.ContainsFunc(s, unicode.IsControl) {
		return errors.New("A name has no control characters")
	}
	return nil
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
