package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vulcanshen/hostu/internal/hosts"
)

func TestSaveAndLoad(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hostu") // not there yet: Save makes it
	s, found, err := Load(dir)
	if err != nil || found || len(s.Profiles) != 0 {
		t.Fatalf("an empty directory: %+v %v %v", s, found, err)
	}
	s.Active = "office"
	s.Profiles = []*Profile{
		{Name: "office", Domains: []hosts.Domain{{Name: "localhost", Addresses: []string{"127.0.0.1", "::1"}}}},
		{Name: "home"},
	}
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, File))
	if !strings.Contains(string(b), "  - name: office\n    domains:\n      - domain: localhost\n        addresses: [127.0.0.1, '::1']") {
		t.Fatalf("profiles.yaml is\n%s", b)
	}
	back, found, err := Load(dir)
	if err != nil || !found || back.Active != "office" || len(back.Profiles) != len(s.Profiles) {
		t.Fatalf("read back %+v %v %v", back, found, err)
	}
	for i, p := range back.Profiles {
		if p.Name != s.Profiles[i].Name || !hosts.Same(p.Domains, s.Profiles[i].Domains) {
			t.Fatalf("read back %+v, wrote %+v", p, s.Profiles[i])
		}
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Fatalf("Save left %d files in the directory", len(entries))
	}
}

func TestLoadBroken(t *testing.T) {
	dir := t.TempDir()
	for _, text := range []string{"profiles: [\n", "active: a\nprofiles:\n  - name: a\n    domains: 3\n"} {
		os.WriteFile(filepath.Join(dir, File), []byte(text), 0o644)
		if s, found, err := Load(dir); err == nil || !found || len(s.Profiles) != 0 || s.Active != "" {
			t.Fatalf("%q: %+v %v %v", text, s, found, err)
		}
	}
}

func TestTakenAndRemove(t *testing.T) {
	office, home := &Profile{Name: "Office"}, &Profile{Name: "home"}
	s := &Store{Profiles: []*Profile{office, home}}
	if !s.Taken("office", nil) || s.Taken("office", office) || s.Taken("lab", nil) {
		t.Fatal("Taken is wrong")
	}
	if s.Get("home") != home || s.Get("Home") != nil {
		t.Fatal("Get is wrong")
	}
	s.Remove(office)
	if len(s.Profiles) != 1 || s.Profiles[0] != home {
		t.Fatalf("left %v", s.Profiles)
	}
}

// A hand-edited profiles.yaml is held to what hostu writes itself: nothing
// the screen cannot show or the hosts file should not have gets in.
func TestLoadChecksWhatIsInIt(t *testing.T) {
	for text, want := range map[string]string{
		"profiles:\n  - name: \"a\\x1b[31m\"\n": "no control characters",
		"profiles:\n  - name: \" a\"\n":         `profile 1 is named " a": a name has no space at either end`,
		"profiles:\n  - name: a\n  - name: A\n": "two profiles are named",
		"profiles:\n  - name: a\n    domains:\n      - domain: \"x\\x1b[31m\"\n        addresses: [10.0.0.1]\n":                                      "which is not a domain",
		"profiles:\n  - name: a\n    domains:\n      - domain: x\n        addresses: [10.0.0]\n":                                                     "not an IPv4 or IPv6 address",
		"profiles:\n  - name: a\n    domains:\n      - domain: x\n":                                                                                  "with no address",
		"profiles:\n  - name: a\n    domains:\n      - domain: x\n        addresses: [10.0.0.1]\n      - domain: X\n        addresses: [10.0.0.2]\n": "has X twice",
		"active: b\nprofiles:\n  - name: a\n": `the active profile "b" is not there`,
		// Quoted: Esc on MOVE ASIDE writes it to the terminal as it is.
		"active: \"\\x1b[2J\"\n": `the active profile "\x1b[2J" is not there`,
	} {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, File), []byte(text), 0o644)
		s, found, err := Load(dir)
		if !found || err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", text, err, want)
		}
		if len(s.Profiles) != 0 || s.Active != "" {
			t.Errorf("%q: kept %+v", text, s)
		}
	}
}
