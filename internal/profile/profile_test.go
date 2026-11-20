package profile

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vulcanshen/hostu/internal/hosts"
)

func TestSaveAndLoad(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hostu") // not there yet: Save makes it
	s, found, out, err := Load(dir)
	if err != nil || found || len(s.Profiles) != 0 || out != nil {
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
	back, found, out, err := Load(dir)
	if err != nil || !found || back.Active != "office" || len(back.Profiles) != len(s.Profiles) || out != nil {
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
		if s, found, _, err := Load(dir); err == nil || !found || len(s.Profiles) != 0 || s.Active != "" {
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

// A hand-edited profiles.yaml is held to what hostu writes itself: what the
// screen cannot show or the hosts file should not have is taken out, the
// rest kept, and out says what went (tdp E3, entries that do not fit).
func TestLoadTakesOutWhatDoesNotFit(t *testing.T) {
	for text, want := range map[string]struct {
		out  []string
		kept string
	}{
		"profiles:\n  - name: \"a\\x1b[31m\"\n  - name: b\n": {[]string{`profile 1 (name "a\x1b[31m")`}, "b"},
		"profiles:\n  - name: \" a\"\n  - name: \"\"\n":      {[]string{`profile 1 (name " a")`, `profile 2 (name "")`}, ""},
		"profiles:\n  - name: a\n  - name: A\n":              {[]string{`profile 2 (name "A" again)`}, "a"},
		"profiles:\n  - name: a\n    domains:\n      - domain: \"x\\x1b[31m\"\n        addresses: [10.0.0.1]\n": {
			[]string{`domain "x\x1b[31m" in the profile a`}, "a:"},
		"profiles:\n  - name: a\n    domains:\n      - domain: x\n        addresses: [10.0.0, 10.0.0.1]\n": {
			[]string{`address "10.0.0" of x in the profile a`}, "a:x=10.0.0.1"},
		"profiles:\n  - name: a\n    domains:\n      - domain: x\n        addresses: [10.0.0]\n": {
			[]string{`address "10.0.0" of x in the profile a`, "domain x in the profile a, with no address"}, "a:"},
		"profiles:\n  - name: a\n    domains:\n      - domain: x\n": {[]string{"domain x in the profile a, with no address"}, "a:"},
		"profiles:\n  - name: a\n    domains:\n      - domain: x\n        addresses: [10.0.0.1]\n      - domain: X\n        addresses: [10.0.0.2]\n": {
			[]string{"domain X in the profile a again"}, "a:x=10.0.0.1"},
	} {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, File), []byte(text), 0o644)
		s, found, out, err := Load(dir)
		if !found || err != nil || !slices.Equal(out, want.out) {
			t.Errorf("%q: %q %v, want %q", text, out, err, want.out)
		}
		var kept []string
		for _, p := range s.Profiles {
			k := p.Name
			if len(p.Domains) > 0 || strings.Contains(want.kept, ":") {
				k += ":"
				for _, d := range p.Domains {
					k += d.Name + "=" + strings.Join(d.Addresses, ",")
				}
			}
			kept = append(kept, k)
		}
		if strings.Join(kept, " ") != want.kept {
			t.Errorf("%q: kept %q, want %q", text, kept, want.kept)
		}
	}
}

// An active profile that is not there is kept as written, not taken out:
// the file is not the place to change it (tdp E3).
func TestActiveGone(t *testing.T) {
	for text, gone := range map[string]bool{"active: b\nprofiles:\n  - name: a\n": true,
		"active: \"\\x1b[2J\"\n": true, "active: a\nprofiles:\n  - name: a\n": false, "profiles:\n  - name: a\n": false} {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, File), []byte(text), 0o644)
		s, _, out, err := Load(dir)
		if err != nil || out != nil || s.ActiveGone() != gone {
			t.Errorf("%q: %+v %q %v", text, s, out, err)
		}
	}
}
