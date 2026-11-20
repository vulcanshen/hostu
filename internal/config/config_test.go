package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	if c, m, n, err := Load(dir); err != nil || c != Default || m != nil || n != nil {
		t.Fatalf("no file: %+v %v %v %v", c, m, n, err)
	}
	for text, want := range map[string]int{"input_tab_width: 2\n": 2, "# nothing set\n": 4, "": 4} {
		os.WriteFile(filepath.Join(dir, File), []byte(text), 0o644)
		if c, m, n, err := Load(dir); err != nil || c.InputTabWidth != want || m != nil || n != nil {
			t.Fatalf("%q: %+v %v %v %v", text, c, m, n, err)
		}
	}
	for _, text := range []string{"input_tab_width: [\n", "- input_tab_width\n"} {
		os.WriteFile(filepath.Join(dir, File), []byte(text), 0o644)
		if c, _, _, err := Load(dir); err == nil || c != Default {
			t.Fatalf("%q: %+v %v", text, c, err)
		}
	}
}

// A value hostu can't use is a misfit, its default used; a key hostu
// doesn't read is told, not a misfit (tdp E3).
func TestLoadMisfitsAndKeysNotRead(t *testing.T) {
	dir := t.TempDir()
	for text, was := range map[string]string{"input_tab_width: 0\n": "0", "input_tab_width: wide\n": "wide",
		"input_tab_width: [2]\n": "a list"} {
		os.WriteFile(filepath.Join(dir, File), []byte(text), 0o644)
		c, m, n, err := Load(dir)
		if err != nil || c != Default || !slices.Equal(m, []Misfit{{"input_tab_width", was, "4"}}) || n != nil {
			t.Fatalf("%q: %+v %v %v %v", text, c, m, n, err)
		}
	}
	os.WriteFile(filepath.Join(dir, File), []byte("refesh: 3\ninput_tab_width: 2\n"), 0o644)
	if c, m, n, err := Load(dir); err != nil || c.InputTabWidth != 2 || m != nil || !slices.Equal(n, []string{"refesh"}) {
		t.Fatalf("%+v %v %v %v", c, m, n, err)
	}
}

// Fix puts the defaults in for the misfits and keeps the rest, comments and
// keys hostu doesn't read too.
func TestFix(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, File), []byte("# mine\nrefesh: 3\ninput_tab_width: -1\n"), 0o600)
	_, m, _, _ := Load(dir)
	if err := Fix(dir, m); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, File))
	if string(b) != "# mine\nrefesh: 3\ninput_tab_width: 4\n" {
		t.Fatalf("%q", b)
	}
	if info, _ := os.Stat(filepath.Join(dir, File)); info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v", info.Mode())
	}
	if c, m, n, err := Load(dir); err != nil || c != Default || m != nil || len(n) != 1 {
		t.Fatalf("%+v %v %v %v", c, m, n, err)
	}
}
