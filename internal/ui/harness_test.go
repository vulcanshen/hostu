package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// systemHosts is the hosts file every test starts from, as a system has it.
const systemHosts = `##
# Host Database
##
127.0.0.1	localhost
255.255.255.255	broadcasthost
::1             localhost
`

// officeAndHome is a profiles.yaml with default active and two more: office
// has no localhost.
const officeAndHome = `active: default
profiles:
  - name: default
    domains:
      - domain: localhost
        addresses: [127.0.0.1, '::1']
      - domain: broadcasthost
        addresses: [255.255.255.255]
  - name: office
    domains:
      - domain: git.corp.local
        addresses: [10.1.2.3, 'fd00::3']
  - name: home
    domains:
      - domain: localhost
        addresses: [127.0.0.1]
      - domain: nas.home
        addresses: [192.168.1.20]
`

// harness runs hostu without a terminal: keys in, the frame out as text.
type harness struct {
	t    *testing.T
	h    *hostu
	env  Env
	quit bool // the program asked to quit
}

// TestMain makes an icon and a powerline symbol one cell each; a test that
// changes them does it while no other test runs.
func TestMain(m *testing.M) {
	terminu.SetIconWidth(1, 1)
	os.Exit(m.Run())
}

// sandbox is an Env in a new temporary directory: the hosts file a system
// has, and profiles.yaml when profiles is not "". The directory has a short
// path, as /etc/hosts is: confirms and toasts that name it fit in a line.
func sandbox(t *testing.T, profiles string) Env {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "hu")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	env := Env{Config: filepath.Join(dir, "config"), Data: filepath.Join(dir, "data"), Hosts: filepath.Join(dir, "hosts")}
	fill(t, env, profiles)
	return env
}

// fill puts the files of a start in env, as sandbox makes them.
func fill(t *testing.T, env Env, profiles string) {
	t.Helper()
	os.RemoveAll(env.Config)
	os.RemoveAll(env.Data)
	if err := os.WriteFile(env.Hosts, []byte(systemHosts), 0o644); err != nil {
		t.Fatal(err)
	}
	if profiles != "" {
		os.MkdirAll(env.Data, 0o755)
		if err := os.WriteFile(filepath.Join(env.Data, "profiles.yaml"), []byte(profiles), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// newHarness is hostu on a w × h screen, its profiles officeAndHome.
func newHarness(t *testing.T, w, h int) *harness {
	t.Helper()
	return start(t, sandbox(t, officeAndHome), w, h)
}

// start is hostu with env on a w × h screen, once the program runs.
func start(t *testing.T, env Env, w, h int) *harness {
	t.Helper()
	hs := &harness{t: t, h: newHostu(env), env: env}
	hs.send(tea.WindowSizeMsg{Width: w, Height: h})
	hs.run(hs.h.a.Init())
	return hs
}

// send gives msg to the program and runs what it sets off, animations
// included, until nothing is left to run.
func (hs *harness) send(msg tea.Msg) {
	_, cmd := hs.h.a.Update(msg)
	hs.run(cmd)
}

// longTimer is how long a command may take before the harness stops waiting
// for it: an animation frame is 32ms, a toast's time on the screen seconds.
const longTimer = 200 * time.Millisecond

func (hs *harness) run(cmd tea.Cmd) {
	deadline := time.Now().Add(3 * time.Second)
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 && time.Now().Before(deadline) {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		done := make(chan tea.Msg, 1)
		go func() { done <- c() }()
		var msg tea.Msg
		select {
		case msg = <-done:
		case <-time.After(longTimer):
			continue
		}
		switch msg := msg.(type) {
		case nil:
		case tea.BatchMsg:
			queue = append(queue, msg...)
		case tea.QuitMsg:
			hs.quit = true
			return
		default:
			_, next := hs.h.a.Update(msg)
			queue = append(queue, next)
		}
	}
}

// keys presses each key: a name as terminu.KeyName writes it ("Enter",
// "Esc", "Space", "Tab", "←"), or text to type.
func (hs *harness) keys(keys ...string) {
	for _, k := range keys {
		hs.send(keyMsg(k))
	}
}

func keyMsg(k string) tea.KeyMsg {
	named := map[string]tea.KeyType{
		"Enter": tea.KeyEnter, "Esc": tea.KeyEsc, "Tab": tea.KeyTab, "Shift-Tab": tea.KeyShiftTab,
		"Space": tea.KeySpace, "Backspace": tea.KeyBackspace,
		"←": tea.KeyLeft, "→": tea.KeyRight, "↑": tea.KeyUp, "↓": tea.KeyDown, "Ctrl-C": tea.KeyCtrlC,
	}
	if t, ok := named[k]; ok {
		if t == tea.KeySpace {
			return tea.KeyMsg{Type: t, Runes: []rune{' '}}
		}
		return tea.KeyMsg{Type: t}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

// frame is the screen as plain text.
func (hs *harness) frame() []string {
	var out []string
	for _, l := range hs.h.a.Frame() {
		out = append(out, terminu.StripStyles(l))
	}
	return out
}

func (hs *harness) text() string { return strings.Join(hs.frame(), "\n") }

// contains fails unless the screen shows every s.
func (hs *harness) contains(s ...string) {
	hs.t.Helper()
	for _, s := range s {
		if !strings.Contains(hs.text(), s) {
			hs.t.Fatalf("the screen does not show %q:\n%s", s, hs.text())
		}
	}
}

func (hs *harness) lacks(s ...string) {
	hs.t.Helper()
	for _, s := range s {
		if strings.Contains(hs.text(), s) {
			hs.t.Fatalf("the screen shows %q:\n%s", s, hs.text())
		}
	}
}

// fullWidth checks tdp L4: every line exactly as wide as the terminal.
func (hs *harness) fullWidth(what string) {
	hs.t.Helper()
	w, h := hs.h.a.Size()
	lines := hs.h.a.Frame()
	if len(lines) != h {
		hs.t.Fatalf("%s: %d lines on a screen %d high", what, len(lines), h)
	}
	for i, l := range lines {
		if got := terminu.Width(l); got != w {
			hs.t.Fatalf("%s: line %d is %d cells on a screen %d wide: %q", what, i, got, w, terminu.StripStyles(l))
		}
	}
}

// hostsFile is the hosts file as it is now.
func (hs *harness) hostsFile() string {
	hs.t.Helper()
	b, err := os.ReadFile(hs.env.Hosts)
	if err != nil {
		hs.t.Fatal(err)
	}
	return string(b)
}

// saved is profiles.yaml as it is now.
func (hs *harness) saved() string {
	b, _ := os.ReadFile(filepath.Join(hs.env.Data, "profiles.yaml"))
	return string(b)
}
