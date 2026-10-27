package hosts

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// stand puts fakes in place of the system for one test: which tools are
// installed, and what running them does. It records every run.
func stand(t *testing.T, system string, installed []string, do func(name string, args []string) error) *[][]string {
	t.Helper()
	var runs [][]string
	oldOut, oldLook, oldGOOS := output, lookPath, goos
	t.Cleanup(func() { output, lookPath, goos = oldOut, oldLook, oldGOOS })
	goos = system
	lookPath = func(n string) (string, error) {
		if slices.Contains(installed, n) {
			return "/usr/bin/" + n, nil
		}
		return "", errors.New("not found")
	}
	output = func(name string, args ...string) ([]byte, error) {
		runs = append(runs, append([]string{name}, args...))
		return nil, do(name, args)
	}
	return &runs
}

// readOnly is a hosts file hostu may not write.
func readOnly(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(path, []byte("127.0.0.1 localhost\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	if f, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
		f.Close()
		t.Skip("running as root: every file is writable")
	}
	return path
}

func TestWriteItself(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts")
	runs := stand(t, "darwin", nil, func(string, []string) error { return errors.New("no tool should run") })
	if e, err := Write(path, "10.0.0.1  a\n"); e != nil || err != nil {
		t.Fatalf("%v %v", e, err)
	}
	if b, _ := os.ReadFile(path); string(b) != "10.0.0.1  a\n" || len(*runs) > 0 {
		t.Fatalf("wrote %q, ran %v", b, *runs)
	}
}

// Elsewhere than Windows the copy is handed back, to run as the administrator
// with sudo, or doas when there is no sudo; Write runs neither (tdp
// terminal.7).
func TestWriteHandsTheCopyBack(t *testing.T) {
	path := readOnly(t)
	for _, c := range []struct {
		installed []string
		tool      string
	}{{[]string{"sudo", "doas"}, "sudo"}, {[]string{"doas"}, "doas"}} {
		runs := stand(t, "darwin", c.installed, func(string, []string) error { return errors.New("no tool should run") })
		e, err := Write(path, "x\n")
		if err != nil || e == nil {
			t.Fatalf("%v %v", e, err)
		}
		if e.Tool != c.tool || strings.Join(e.Command, " ") != "cp "+e.Temp+" "+path || len(*runs) > 0 {
			t.Fatalf("with %v the copy is %s %v, ran %v", c.installed, e.Tool, e.Command, *runs)
		}
		if b, _ := os.ReadFile(e.Temp); string(b) != "x\n" {
			t.Fatalf("the temporary file has %q", b)
		}
		e.Done()
		if _, err := os.Stat(e.Temp); err == nil {
			t.Fatal("the temporary file is left")
		}
	}
}

func TestWriteNoSudo(t *testing.T) {
	path := readOnly(t)
	stand(t, "linux", nil, func(string, []string) error { return nil })
	if _, err := Write(path, "x\n"); err == nil || !strings.Contains(err.Error(), "neither sudo nor doas") {
		t.Fatalf("%v", err)
	}
}

// Windows asks in a UAC window: gsudo runs the copy and Write waits for it.
func TestWriteWindows(t *testing.T) {
	path := readOnly(t)
	runs := stand(t, "windows", []string{"gsudo", "sudo"}, func(string, []string) error { return nil })
	if e, err := Write(path, "x\n"); e != nil || err != nil {
		t.Fatalf("%v %v", e, err)
	}
	if len(*runs) != 1 || (*runs)[0][0] != "gsudo" || !strings.Contains((*runs)[0][4], "Copy-Item -LiteralPath '") {
		t.Fatalf("ran %v", *runs)
	}
	stand(t, "windows", nil, nil)
	if _, err := Write(path, "x\n"); err == nil || !strings.Contains(err.Error(), "Install gsudo") {
		t.Fatalf("%v", err)
	}
}

// A copy that fails says why in the line of PowerShell's output that does;
// all of it is kept for the App Log (tdp components.5).
func TestWriteWindowsFails(t *testing.T) {
	path := readOnly(t)
	stand(t, "windows", []string{"gsudo"}, nil)
	output = func(string, ...string) ([]byte, error) {
		return []byte("Copy-Item : Access to the path is denied.\r\nAt line:1 char:1\r\n\r\n+ Copy-Item -LiteralPath 'x'\r\n"), errors.New("exit status 1")
	}
	_, err := Write(path, "x\n")
	var te *ToolError
	if !errors.As(err, &te) || err.Error() != "Could not write it with gsudo: Copy-Item : Access to the path is denied." {
		t.Fatalf("%v", err)
	}
	if te.Output != "Copy-Item : Access to the path is denied. At line:1 char:1 + Copy-Item -LiteralPath 'x'" {
		t.Fatalf("output %q", te.Output)
	}
	output = func(string, ...string) ([]byte, error) { return nil, errors.New("exit status 1") }
	if _, err := Write(path, "x\n"); err == nil || err.Error() != "Could not write it with gsudo: exit status 1" {
		t.Fatalf("%v", err)
	}
}
