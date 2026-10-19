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
	oldRun, oldOut, oldLook, oldGOOS := run, output, lookPath, goos
	t.Cleanup(func() { run, output, lookPath, goos = oldRun, oldOut, oldLook, oldGOOS })
	goos = system
	lookPath = func(n string) (string, error) {
		if slices.Contains(installed, n) {
			return "/usr/bin/" + n, nil
		}
		return "", errors.New("not found")
	}
	run = func(name string, args ...string) error {
		runs = append(runs, append([]string{name}, args...))
		return do(name, args)
	}
	output = func(name string, args ...string) ([]byte, error) { return nil, run(name, args...) }
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

// sudo without asking: a password typed a moment ago is enough.
func TestWriteSudoWithoutAsking(t *testing.T) {
	path := readOnly(t)
	runs := stand(t, "linux", []string{"sudo", "doas"}, func(string, []string) error { return nil })
	e, err := Write(path, "x\n")
	if e != nil || err != nil {
		t.Fatalf("%v %v", e, err)
	}
	if len(*runs) != 1 || strings.Join((*runs)[0][:3], " ") != "sudo -n cp" || (*runs)[0][4] != path {
		t.Fatalf("ran %v", *runs)
	}
	if _, err := os.Stat((*runs)[0][3]); err == nil {
		t.Fatal("the temporary file is left")
	}
}

// sudo that wants a password: the copy is handed back, to run in a terminal.
func TestWriteSudoWantsAPassword(t *testing.T) {
	path := readOnly(t)
	stand(t, "darwin", []string{"doas"}, func(string, []string) error { return errors.New("a password is required") })
	e, err := Write(path, "x\n")
	if err != nil || e == nil {
		t.Fatalf("%v %v", e, err)
	}
	defer e.Done()
	if strings.Join(e.Command, " ") != "doas cp "+e.Temp+" "+path {
		t.Fatalf("the copy is %v", e.Command)
	}
	if b, _ := os.ReadFile(e.Temp); string(b) != "x\n" {
		t.Fatalf("the temporary file has %q", b)
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
