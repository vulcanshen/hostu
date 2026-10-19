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
	os.WriteFile(filepath.Join(dir, File), []byte("profiles: [\n"), 0o644)
	if _, found, err := Load(dir); err == nil || !found {
		t.Fatalf("a broken file: %v %v", found, err)
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
