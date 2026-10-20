package hosts

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Elevated is a copy that waits for a password: Command copies Temp over the
// hosts file as the administrator, to be run in a terminal where sudo or
// doas can ask. Done removes Temp once it has run.
type Elevated struct {
	Command []string
	Temp    string
}

// Done removes the temporary file.
func (e *Elevated) Done() { os.Remove(e.Temp) }

// run and lookPath are exec's, for the tests to stand in for.
var (
	run      = func(name string, args ...string) error { return exec.Command(name, args...).Run() }
	output   = func(name string, args ...string) ([]byte, error) { return exec.Command(name, args...).CombinedOutput() }
	lookPath = exec.LookPath
	goos     = runtime.GOOS
)

// Write replaces the hosts file at path with text. It writes the file itself
// when it may. Otherwise the text goes in a temporary file that is copied
// over the hosts file as the administrator: on Windows by gsudo or sudo,
// which ask in a window of their own (UAC); elsewhere by sudo or doas without
// asking (-n), and when they want a password Write returns the copy to run
// in a terminal.
func Write(path, text string) (*Elevated, error) {
	err := os.WriteFile(path, []byte(text), 0o644)
	if err == nil || !errors.Is(err, fs.ErrPermission) {
		return nil, err
	}
	tmp, err := os.CreateTemp("", "hostu-hosts-*")
	if err != nil {
		return nil, err
	}
	_, err = tmp.WriteString(text)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp.Name())
		return nil, err
	}
	if goos == "windows" {
		defer os.Remove(tmp.Name())
		return nil, windowsCopy(tmp.Name(), path)
	}
	tool := onPath("sudo", "doas")
	if tool == "" {
		os.Remove(tmp.Name())
		return nil, fmt.Errorf("No permission to write %s, and neither sudo nor doas is there to write it as the administrator.", path)
	}
	if run(tool, "-n", "cp", tmp.Name(), path) == nil {
		os.Remove(tmp.Name())
		return nil, nil
	}
	return &Elevated{Command: []string{tool, "cp", tmp.Name(), path}, Temp: tmp.Name()}, nil
}

// windowsCopy copies tmp over path as the administrator, by gsudo or by the
// sudo Windows 11 has, either of which asks in a UAC window.
func windowsCopy(tmp, path string) error {
	tool := onPath("gsudo", "sudo")
	if tool == "" {
		return fmt.Errorf("No permission to write %s. Install gsudo (scoop install gsudo), or open the terminal as the administrator.", path)
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	if out, err := output(tool, "powershell", "-NoProfile", "-Command",
		"Copy-Item -LiteralPath "+quote(tmp)+" -Destination "+quote(path)+" -Force"); err != nil {
		return fmt.Errorf("Could not write it with %s: %s", tool, strings.TrimSpace(string(out)))
	}
	return nil
}

// onPath is the first of names that is installed, "" for none.
func onPath(names ...string) string {
	for _, n := range names {
		if _, err := lookPath(n); err == nil {
			return n
		}
	}
	return ""
}
