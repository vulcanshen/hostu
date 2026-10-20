package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	if c, err := Load(dir); err != nil || c != Default {
		t.Fatalf("no file: %+v %v", c, err)
	}
	for text, want := range map[string]int{"input_tab_width: 2\n": 2, "# nothing set\n": 4} {
		os.WriteFile(filepath.Join(dir, File), []byte(text), 0o644)
		if c, err := Load(dir); err != nil || c.InputTabWidth != want {
			t.Fatalf("%q: %+v %v", text, c, err)
		}
	}
	for _, text := range []string{"input_tab_width: 0\n", "input_tab_width: [\n"} {
		os.WriteFile(filepath.Join(dir, File), []byte(text), 0o644)
		if c, err := Load(dir); err == nil || c != Default {
			t.Fatalf("%q: %+v %v", text, c, err)
		}
	}
}

// The new config.yaml in place of one moved aside reads as the defaults,
// in a directory not there yet too.
func TestWriteDefault(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "hostu")
	if err := WriteDefault(dir); err != nil {
		t.Fatal(err)
	}
	if c, err := Load(dir); err != nil || c != Default {
		t.Fatalf("%+v %v", c, err)
	}
}
