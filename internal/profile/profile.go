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
// file that can't be read gives a store with no profiles and the error:
// nothing of it is shown or written. One that can is held to what hostu
// writes itself, out saying what of it was taken out (fit); its active
// profile is as the file has it, there or not (tdp E3).
func Load(dir string) (s *Store, found bool, out []string, err error) {
	path := filepath.Join(dir, File)
	s = &Store{path: path}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, false, nil, nil
	}
	if err != nil {
		return s, false, nil, err
	}
	if err := yaml.Unmarshal(b, s); err != nil {
		return &Store{path: path}, true, nil, err
	}
	s.Profiles = slices.DeleteFunc(s.Profiles, func(p *Profile) bool { return p == nil })
	return s, true, s.fit(), nil
}

// fit takes out of a hand-edited store what hostu would not write itself,
// so that no name it can't show and no line it shouldn't write gets in: a
// profile not named as CheckName has it, empty or as one before it; a
// domain a hosts file doesn't take, in its profile twice or with no address
// left; an address that is not one. out says what each was (tdp E3, entries
// that do not fit).
func (s *Store) fit() (out []string) {
	var kept []*Profile
	for i, p := range s.Profiles {
		switch {
		case p.Name == "" || CheckName(p.Name) != nil:
			out = append(out, fmt.Sprintf("profile %d (name %q)", i+1, p.Name))
			continue
		case slices.ContainsFunc(kept, func(k *Profile) bool { return strings.EqualFold(k.Name, p.Name) }):
			out = append(out, fmt.Sprintf("profile %d (name %q again)", i+1, p.Name))
			continue
		}
		var domains []hosts.Domain
		for _, d := range p.Domains {
			switch {
			case hosts.CheckDomain(d.Name) != nil:
				out = append(out, fmt.Sprintf("domain %q in the profile %s", d.Name, p.Name))
				continue
			case hosts.Has(domains, d.Name):
				out = append(out, fmt.Sprintf("domain %s in the profile %s again", d.Name, p.Name))
				continue
			}
			var addrs []string
			for _, a := range d.Addresses {
				if hosts.CheckAddress(a) != nil {
					out = append(out, fmt.Sprintf("address %q of %s in the profile %s", a, d.Name, p.Name))
					continue
				}
				addrs = append(addrs, a)
			}
			if len(addrs) == 0 {
				out = append(out, fmt.Sprintf("domain %s in the profile %s, with no address", d.Name, p.Name))
				continue
			}
			domains = append(domains, hosts.Domain{Name: d.Name, Addresses: addrs})
		}
		p.Domains = domains
		kept = append(kept, p)
	}
	s.Profiles = kept
	return out
}

// ActiveGone reports that the active profile the store names is not in it
// (tdp E3, a value naming an entry that is not there).
func (s *Store) ActiveGone() bool { return s.Active != "" && s.Get(s.Active) == nil }

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
