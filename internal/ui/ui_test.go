package ui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/hostu/internal/hosts"
)

// states are the keys from the start to every screen and every popup, or
// popups over popups, hostu opens. The screens come first, then the popups,
// then the splash.
var states = []struct {
	name string
	keys []string
}{
	{"PROFILES", nil},
	{"[2] focused", []string{"Tab"}},
	{"Space menu on [1]", []string{"Space"}},
	{"Space menu on [2]", []string{"Tab", "Space"}},
	{"global operation popup over the Space menu", []string{"Space", "G", "Enter"}},
	{"App Log", []string{"!"}},
	{"key reference", []string{"?"}},
	{"activate confirm", []string{"Enter"}},
	{"activate confirm with a warning", []string{"j", "Enter"}},
	{"preview", []string{"p"}},
	{"rename", []string{"r"}},
	{"copy", []string{"c"}},
	{"new profile", []string{"n"}},
	{"new profile, a name taken", []string{"n", "home", "Enter"}},
	{"import", []string{"i"}},
	{"delete profile confirm", []string{"j", "D"}},
	{"new domain form", []string{"Tab", "n"}},
	{"edit form", []string{"Tab", "Enter"}},
	{"domain text popup over the form", []string{"Tab", "n", "Enter"}},
	{"addresses, one wrong", []string{"Tab", "n", "j", "Enter", "10.0.0.1 10.0", "Enter"}},
	{"form errors", []string{"Tab", "n", "Tab", "Enter"}},
	{"delete domain confirm", []string{"Tab", "D"}},
	{"splash", []string{"V"}},
}

// ownPopups are the popups hostu opens by what happens rather than by a key:
// the toasts, an error popup, MOVE ASIDE, sudo's terminal popup, the App Log
// with an entry of each level. env, when set, is where
// hostu starts instead of the officeAndHome sandbox; then runs once it has
// started.
var ownPopups = []struct {
	name string
	env  func(t *testing.T) Env
	then func(hs *harness)
}{
	{"the first start's INFO toast", func(t *testing.T) Env { return sandbox(t, "") }, nil},
	{"a SUCCESS toast", nil, func(hs *harness) { hs.keys("Enter", "Enter") }},
	{"a WARNING toast", func(t *testing.T) Env {
		env := sandbox(t, officeAndHome)
		os.WriteFile(env.Hosts, []byte(systemHosts+"not an address\n"), 0o644)
		return env
	}, func(hs *harness) { hs.keys("i", "Enter") }},
	{"an error popup", func(t *testing.T) Env {
		env := sandbox(t, officeAndHome)
		os.Remove(env.Hosts)
		os.Mkdir(env.Hosts, 0o755) // a directory: it can't be read
		return env
	}, nil},
	{"MOVE ASIDE", func(t *testing.T) Env { return sandbox(t, "profiles: [\n") }, nil},
	{"the App Log", func(t *testing.T) Env {
		env := sandbox(t, "")
		os.WriteFile(env.Hosts, []byte(systemHosts+"not an address\n"), 0o644)
		return env
	}, func(hs *harness) { // a WARN of the first start, an OK of a new profile
		hs.keys("n", "lab", "Enter")
		hs.h.a.Log(terminu.Info, "an info")
		hs.h.a.Log(terminu.Error, "an error")
		hs.keys("!")
	}},
	{"sudo's terminal popup", nil, func(hs *harness) {
		hs.send(written{p: hs.h.store.Get("home"),
			elevated: &hosts.Elevated{Command: []string{"sh", "-c", "sleep 2"}, Temp: filepath.Join(hs.env.Data, "none")}})
	}},
}

// iconWidths are the widths of an icon and a powerline symbol the screen
// tests run with (tdp E5).
var iconWidths = [][2]int{{1, 1}, {2, 1}, {2, 2}}

// eachIconWidth runs check in parallel for every case, once for each of
// iconWidths. Only the widths change between the rounds, while nothing runs.
func eachIconWidth[C any](t *testing.T, cases []C, name func(C) string, check func(*testing.T, C)) {
	defer terminu.SetIconWidth(1, 1)
	for _, iw := range iconWidths {
		terminu.SetIconWidth(iw[0], iw[1])
		t.Run(fmt.Sprintf("icon %d, powerline %d", iw[0], iw[1]), func(t *testing.T) {
			for _, c := range cases {
				t.Run(name(c), func(t *testing.T) {
					t.Parallel()
					check(t, c)
				})
			}
		})
	}
}

type sized struct {
	w, h  int
	name  string
	keys  []string
	small [2]int // the size to shrink to once it is open; 0 for none
	env   func(t *testing.T) Env
	then  func(hs *harness)
}

// TestL4EveryLineIsTheTerminalsWidth is tdp L4: on every screen and under
// every popup, at many sizes from the least hostu needs up. hostu measures
// its whole screen; each popup's own frame is the lib's to measure, the lib
// draws it (tdp E8).
func TestL4EveryLineIsTheTerminalsWidth(t *testing.T) {
	var cases []sized
	for _, sz := range [][2]int{{60, 24}, {60, 32}, {80, 24}, {120, 40}, {160, 50}} {
		for _, st := range states {
			cases = append(cases, sized{w: sz[0], h: sz[1], name: st.name, keys: st.keys})
		}
		for _, p := range ownPopups {
			cases = append(cases, sized{w: sz[0], h: sz[1], name: p.name, env: p.env, then: p.then})
		}
	}
	eachIconWidth(t, cases, func(c sized) string { return fmt.Sprintf("%dx%d %s", c.w, c.h, c.name) }, checkSized)
}

// TestE5PopupBiggerThanTheScreen is tdp E5: a popup keeps the size it opened
// with (tdp F7), so shrinking the terminal leaves it wider than the
// screen, taller, or both; every line still fits. Below the least size, to
// 1×1, only the name is left and nothing panics (tdp L1).
func TestE5PopupBiggerThanTheScreen(t *testing.T) {
	var cases []sized
	for _, st := range states[2:] { // the popups
		for _, small := range [][2]int{{60, 32}, {120, 24}, {60, 24}, {60, 5}, {1, 1}} {
			cases = append(cases, sized{w: 120, h: 40, name: st.name, keys: st.keys, small: small})
		}
	}
	for _, p := range ownPopups {
		for _, small := range [][2]int{{60, 32}, {120, 24}, {60, 24}, {60, 5}, {1, 1}} {
			cases = append(cases, sized{w: 120, h: 40, name: p.name, env: p.env, then: p.then, small: small})
		}
	}
	eachIconWidth(t, cases, func(c sized) string { return fmt.Sprintf("%s, then %dx%d", c.name, c.small[0], c.small[1]) }, checkSized)
}

func checkSized(t *testing.T, c sized) {
	var hs *harness
	if c.env != nil {
		hs = start(t, c.env(t), c.w, c.h)
	} else {
		hs = newHarness(t, c.w, c.h)
	}
	if c.then != nil {
		c.then(hs)
	}
	hs.keys(c.keys...)
	if c.small[0] > 0 {
		hs.send(tea.WindowSizeMsg{Width: c.small[0], Height: c.small[1]})
	}
	hs.fullWidth(c.name)
}

// TestL1BelowTheLeastSize is tdp L1: smaller than 60×24 either way, only
// hostu's name shows, under it the size it needs and how to quit when they
// fit, the size going first. No key acts but Ctrl-C twice, and all comes
// back as the terminal grows.
func TestL1BelowTheLeastSize(t *testing.T) {
	const hint, quit = "needs a terminal of at least 60×24", "Press [Ctrl-C] twice to quit"
	for _, sz := range [][2]int{{59, 24}, {60, 23}, {40, 12}, {30, 5}, {60, 2}, {60, 1}, {1, 1}} {
		t.Run(fmt.Sprintf("%dx%d", sz[0], sz[1]), func(t *testing.T) {
			t.Parallel()
			hs := newHarness(t, 80, 40)
			hs.keys("Space") // a menu open, to come back to
			small := tea.WindowSizeMsg{Width: sz[0], Height: sz[1]}
			hs.send(small)
			hs.fullWidth("below the least size")
			if sz[0] >= len("hostu") {
				hs.contains("hostu")
			}
			if sz[0] >= terminu.Width(hint) && sz[1] >= 3 {
				hs.contains(hint)
			} else {
				hs.lacks(hint)
			}
			if sz[0] >= terminu.Width(quit) && sz[1] >= 2 {
				hs.contains(quit)
			} else {
				hs.lacks(quit)
			}
			hs.lacks("ITEM OPERATION")
			hs.keys("q", "Esc", "Enter", "Enter", "Ctrl-C")
			if hs.quit {
				t.Fatal("a key quit below the least size")
			}
			hs.send(tea.WindowSizeMsg{Width: 80, Height: 40})
			hs.contains("ITEM OPERATION", "[Enter] Activate")
			hs.lacks("ACTIVATE: default")
			if hs.hostsFile() != systemHosts {
				t.Fatal("a key acted below the least size")
			}
			hs.send(small)
			hs.keys("Ctrl-C", "Ctrl-C")
			if !hs.quit {
				t.Fatal("Ctrl-C twice did not quit below the least size")
			}
		})
	}
}

// The same profiles, in the same order, every time hostu starts.
func TestSameEveryStart(t *testing.T) {
	for _, st := range states {
		t.Run(st.name, func(t *testing.T) {
			t.Parallel()
			env := sandbox(t, officeAndHome)
			a := start(t, env, 80, 40)
			a.keys(st.keys...)
			at := a.text()
			fill(t, env, officeAndHome)
			b := start(t, env, 80, 40)
			b.keys(st.keys...)
			if bt := b.text(); at != bt {
				t.Fatalf("it differs between two starts:\n%s\n\n%s", at, bt)
			}
		})
	}
}

// TestSplashIsDocsIcon is tdp E6: the splash draws docs/icon.svg, hostu's
// mark as tdp S5 has it, the U frame and then H, O, S, T, U.
func TestSplashIsDocsIcon(t *testing.T) {
	t.Parallel()
	want, err := os.ReadFile("../../docs/icon.svg")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(icon, want) {
		t.Fatal("internal/ui/icon.svg is not docs/icon.svg: copy docs/icon.svg over it")
	}
	var comments []string
	for _, l := range strings.Split(string(icon), "\n") {
		if c, ok := strings.CutPrefix(strings.TrimSpace(l), "<!-- "); ok {
			comments = append(comments, strings.TrimSuffix(c, " -->"))
		}
	}
	if !slices.Equal(comments, []string{"U", "H", "O", "S", "T", "U"}) {
		t.Fatalf("the letters are %q, want U H O S T U", comments)
	}
	s := splash()
	var lines []string
	for f := 0; ; f++ {
		var done bool
		if lines, done = s(f, 120, 40); done {
			break
		}
		if f > 1000 {
			t.Fatal("the reveal never ends")
		}
	}
	text := terminu.StripStyles(strings.Join(lines, "\n"))
	for _, want := range []string{"hostu", "dev", "Switch the hosts file between profiles", "Press any key to close"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the splash does not show %q:\n%s", want, text)
		}
	}
}

// The first start reads the hosts file into default, the active profile.
func TestFirstStartMakesDefault(t *testing.T) {
	t.Parallel()
	hs := start(t, sandbox(t, ""), 80, 40)
	hs.contains("Made the profile default from", "default", "localhost", "127.0.0.1, ::1", "broadcasthost", "active: default")
	if !strings.Contains(hs.saved(), "active: default") {
		t.Fatalf("profiles.yaml is\n%s", hs.saved())
	}
	if hs.h.differs() {
		t.Fatal("default differs from the hosts file it came from")
	}
	hs = start(t, hs.env, 80, 40) // the second start says nothing
	hs.lacks("Made the profile default")
}

// The lines of the hosts file default leaves out are told in a Warning toast.
func TestFirstStartSaysWhatItSkipped(t *testing.T) {
	t.Parallel()
	env := sandbox(t, "")
	os.WriteFile(env.Hosts, []byte(systemHosts+"not an address\n"), 0o644)
	hs := start(t, env, 120, 40)
	hs.contains("WARNING", "Made the profile default from "+env.Hosts+", skipped 1 line")
}

func TestActivateWritesTheHostsFile(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("G", "Enter") // home
	hs.contains("ACTIVATE: home", "The hosts file gets 2 domains, and nothing else.", "Write home to "+hs.env.Hosts+"?")
	hs.lacks("has no localhost")
	hs.keys("Enter")
	home := hs.h.store.Get("home")
	if hs.hostsFile() != hosts.Format("home", home.Domains) {
		t.Fatalf("the hosts file is\n%s", hs.hostsFile())
	}
	hs.contains("Wrote home to "+hs.env.Hosts, "active: home")
	if !strings.Contains(hs.saved(), "active: home") {
		t.Fatalf("profiles.yaml is\n%s", hs.saved())
	}
	// The hosts file as it was is kept, the first time only.
	orig := filepath.Join(hs.env.Data, original)
	if b, _ := os.ReadFile(orig); string(b) != systemHosts {
		t.Fatalf("hosts.original is %q", b)
	}
	hs.keys("Esc", "gg", "Enter", "Enter") // default
	if b, _ := os.ReadFile(orig); string(b) != systemHosts {
		t.Fatalf("hosts.original changed to %q", b)
	}
	hs.contains("active: default")
}

// Esc on the confirm writes nothing (tdp F6).
func TestActivateEscWritesNothing(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("j", "Enter", "Esc")
	if hs.hostsFile() != systemHosts || hs.h.store.Active != "default" {
		t.Fatal("Esc wrote the hosts file")
	}
}

func TestActivateWarnsWithoutLocalhost(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("j", "Enter")
	hs.contains("office has no localhost: programs that connect to localhost stop working.")
}

// Changing the active profile does not write the hosts file: the statusbar
// says they differ until it is activated again.
func TestStatusSaysTheHostsFileDiffers(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.contains("(in sync)")
	hs.keys("Tab", "j", "D", "Enter") // broadcasthost out of default
	hs.contains("(differs)     active: default")
	if hs.hostsFile() != systemHosts {
		t.Fatal("a change to the profile wrote the hosts file")
	}
	hs.keys("1", "Enter", "Enter")
	hs.contains("(in sync)")
}

func TestNewProfile(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("n", "Enter")
	hs.contains("Name can't be empty")
	hs.keys("Office", "Enter")
	hs.contains("There is a profile named Office already")
	hs.keys("Backspace", "Backspace", "Backspace", "Backspace", "Backspace", "Backspace", "lab", "Enter")
	if r, _ := hs.h.profiles.Current(); r.ID != "lab" {
		t.Fatalf("the cursor is on %s, not the new profile", r.ID)
	}
	hs.contains("DOMAINS: lab", "No domains in lab", "Press [n] to add one")
	if !strings.Contains(hs.saved(), "name: lab") {
		t.Fatalf("profiles.yaml is\n%s", hs.saved())
	}
}

func TestRenameTheActiveProfile(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("r", "system", "Enter")
	if hs.h.store.Active != "system" || hs.h.store.Get("system") == nil {
		t.Fatalf("active %q, profiles %v", hs.h.store.Active, hs.h.store.Profiles)
	}
	hs.contains("active: system", "DOMAINS: system")
}

func TestCopyProfile(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("G", "c")
	hs.contains("COPY: home", "home copy")
	hs.keys("Enter")
	cp := hs.h.store.Get("home copy")
	if cp == nil || !hosts.Same(cp.Domains, hs.h.store.Get("home").Domains) {
		t.Fatalf("the copy is %+v", cp)
	}
	cp.Domains[0].Addresses[0] = "127.0.0.2"
	if hs.h.store.Get("home").Domains[0].Addresses[0] != "127.0.0.1" {
		t.Fatal("the copy shares its addresses with home")
	}
}

func TestImportTheHostsFile(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	os.WriteFile(hs.env.Hosts, []byte(systemHosts+"10.9.9.9 build.corp\nnot an address\n"), 0o644)
	hs.keys("i", "Enter")
	hs.contains("WARNING", "Imported 3 domains from "+hs.env.Hosts+", skipped 1 line")
	p := hs.h.store.Get("imported")
	if p == nil || len(p.Domains) != 3 || p.Domains[2].Name != "build.corp" {
		t.Fatalf("imported %+v", p)
	}
	if r, _ := hs.h.profiles.Current(); r.ID != "imported" {
		t.Fatalf("the cursor is on %s", r.ID)
	}
}

// The active profile is in the hosts file: Delete is disabled on it (tdp M6).
func TestDeleteProfile(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("D")
	hs.lacks("DELETE")
	hs.keys("j", "D")
	hs.contains("DELETE: office", "Its one domain goes with it.", "Delete the profile office?")
	hs.keys("Enter")
	if hs.h.store.Get("office") != nil || strings.Contains(hs.saved(), "office") {
		t.Fatal("office is still there")
	}
}

func TestNewDomain(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("G", "Tab", "n") // home
	hs.contains("NEW DOMAIN", "Domain", "Addresses", "not set", "Add")
	hs.keys("Enter", "git corp", "Enter")
	hs.contains("A domain has only letters, digits, -, _ and ., not a space")
	hs.keys("Esc", "Enter", "NAS.home", "Enter", "j", "Enter", "10.0.0.1, fd00::1", "Enter", "Tab", "Enter")
	hs.contains("NAS.home is in home already")
	hs.keys("Enter", "printer.home", "Enter", "Tab", "Enter") // the focus is on Domain, the field wrong

	hs.lacks("NEW DOMAIN")
	home := hs.h.store.Get("home")
	if d := home.Domains[2]; d.Name != "printer.home" || !slices.Equal(d.Addresses, []string{"10.0.0.1", "fd00::1"}) {
		t.Fatalf("added %+v", d)
	}
	if r, _ := hs.h.domains.Current(); r.ID != "printer.home" {
		t.Fatalf("the cursor is on %s, not the new domain", r.ID)
	}
	hs.contains("printer.home", "10.0.0.1, fd00::1")
}

// Enter on a domain edits it; from the Space menu, the menu stays under the
// form and comes back when it is saved (tdp T1).
func TestEditDomain(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("j", "Tab", "Space", "Enter")
	hs.contains("EDIT: git.corp.local", "10.1.2.3 fd00::3", "Save")
	// Tab takes the old value offered in grey, to change it.
	hs.keys("j", "Enter", "Tab", "Backspace", "Backspace", "Backspace", "Backspace", "Backspace", "Backspace", "Backspace", "Backspace", "Enter", "Tab", "Enter")
	if d := hs.h.store.Get("office").Domains[0]; !slices.Equal(d.Addresses, []string{"10.1.2.3"}) {
		t.Fatalf("saved %+v", d)
	}
	hs.contains("ITEM OPERATION", "[Enter] Edit")
}

func TestDeleteDomain(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("G", "Tab", "j", "D")
	hs.contains("DELETE: nas.home", "Delete nas.home from home?")
	hs.keys("Enter")
	if home := hs.h.store.Get("home"); len(home.Domains) != 1 {
		t.Fatalf("home has %+v", home.Domains)
	}
}

func TestPreview(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("G", "p")
	hs.contains("HOSTS FILE — home", `# Written by hostu from the profile "home".`, "192.168.1.20  nas.home")
}

// broken are profiles.yaml files hostu can't read and why MOVE ASIDE says:
// not YAML, YAML of the wrong shape, YAML that check finds wrong. yaml's own
// "yaml: " is left out, its errors are on one line, and a value of the wrong
// kind is said in the user's words, not Go's.
var broken = map[string]string{
	"profiles: [\n":              "line 1: did not find expected node content",
	"profiles: 3\n":              "line 1: 3 is not a list",
	"active: [a]\nprofiles: 3\n": "line 1: a list is not text; line 2: 3 is not a list",
	"active: office\nprofiles:\n  - name: office\n    domains:\n      - domain: x\n        addresses: [10.0.0]\n": `the profile office has "10.0.0" for x, which is not an IPv4 or IPv6 address`,
}

// A profiles.yaml hostu can't read is asked about before anything else
// (tdp E3): Esc leaves hostu, the file and the hosts file as they were.
func TestBrokenProfilesEscQuits(t *testing.T) {
	t.Parallel()
	for text, why := range broken {
		hs := start(t, sandbox(t, text), 120, 40)
		path := filepath.Join(hs.env.Data, "profiles.yaml")
		hs.contains("MOVE ASIDE", path+" can't be read:", why, "Move it to profiles.yaml.bak and start fresh?", "Enter:move Esc:quit")
		hs.lacks("yaml:")
		hs.keys("Esc")
		if !hs.quit {
			t.Fatalf("%q: Esc did not leave hostu:\n%s", text, hs.text())
		}
		if _, err := os.Stat(path + ".bak"); hs.saved() != text || hs.hostsFile() != systemHosts || err == nil {
			t.Fatalf("%q: profiles.yaml is now %q, the hosts file %q", text, hs.saved(), hs.hostsFile())
		}
	}
}

// Enter puts it aside as profiles.yaml.bak, never over another (.bak.1
// then), and hostu starts as the first time: default from the hosts file.
func TestBrokenProfilesMovedAside(t *testing.T) {
	t.Parallel()
	env := sandbox(t, "profiles: [\n")
	path := filepath.Join(env.Data, "profiles.yaml")
	os.WriteFile(path+".bak", []byte("an older one\n"), 0o644)
	hs := start(t, env, 120, 40)
	hs.contains("Move it to profiles.yaml.bak.1 and start fresh?")
	hs.keys("Enter")
	if b, _ := os.ReadFile(path + ".bak.1"); string(b) != "profiles: [\n" {
		t.Fatalf("profiles.yaml.bak.1 is %q", b)
	}
	if b, _ := os.ReadFile(path + ".bak"); string(b) != "an older one\n" {
		t.Fatalf("profiles.yaml.bak is %q", b)
	}
	hs.lacks("MOVE ASIDE")
	hs.contains("Made the profile default from "+env.Hosts, "active: default", "broadcasthost")
	if !strings.Contains(hs.saved(), "active: default") {
		t.Fatalf("profiles.yaml is\n%s", hs.saved())
	}
	hs.keys("!")
	hs.contains("Moved "+path+" to "+path+".bak.1", "Made the profile default")
}

// A config.yaml hostu can't read is asked about first, then profiles.yaml.
// Enter on the first starts as the first time: the defaults, and no
// config.yaml, which the first start does not make (tdp E3-18).
func TestBrokenConfigThenProfiles(t *testing.T) {
	t.Parallel()
	env := sandbox(t, "profiles: [\n")
	os.MkdirAll(env.Config, 0o755)
	cfg := filepath.Join(env.Config, "config.yaml")
	os.WriteFile(cfg, []byte("input_tab_width: 0\n"), 0o644)
	hs := start(t, env, 120, 40)
	hs.contains(cfg+" can't be read:", "the setting input_tab_width is 0: it is at least 1", "Move it to config.yaml.bak")
	hs.keys("Enter")
	if _, err := os.Stat(cfg); err == nil || hs.h.a.InputTabWidth != 4 {
		t.Fatalf("config.yaml is there (%v) or input_tab_width is %d", err, hs.h.a.InputTabWidth)
	}
	if b, _ := os.ReadFile(cfg + ".bak"); string(b) != "input_tab_width: 0\n" {
		t.Fatalf("config.yaml.bak is %q", b)
	}
	hs.contains(filepath.Join(env.Data, "profiles.yaml")+" can't be read:", "Move it to profiles.yaml.bak")
	hs.keys("Enter")
	hs.lacks("MOVE ASIDE")
	hs.contains("active: default")
	hs.keys("!")
	hs.contains(cfg + " to " + cfg + ".bak")
	hs.lacks("Wrote")
}

// Asked about config.yaml at the first start, hostu has written nothing yet:
// Esc leaves no profiles.yaml behind.
func TestBrokenConfigAtTheFirstStart(t *testing.T) {
	t.Parallel()
	env := sandbox(t, "")
	os.MkdirAll(env.Config, 0o755)
	os.WriteFile(filepath.Join(env.Config, "config.yaml"), []byte("input_tab_width: 0\n"), 0o644)
	hs := start(t, env, 120, 40)
	hs.contains("MOVE ASIDE")
	hs.lacks("Made the profile default")
	hs.keys("Esc")
	if _, err := os.Stat(filepath.Join(env.Data, "profiles.yaml")); !hs.quit || err == nil {
		t.Fatalf("quit %v, profiles.yaml %v", hs.quit, err)
	}
}

// The hosts file is not hostu's own: one it can't read is an error popup,
// hostu goes on, and the statusbar says it.
func TestHostsFileCantBeRead(t *testing.T) {
	t.Parallel()
	env := sandbox(t, officeAndHome)
	os.Remove(env.Hosts)
	os.Mkdir(env.Hosts, 0o755)
	hs := start(t, env, 120, 40)
	hs.contains("CAN'T READ", "(can't read)")
	hs.lacks("MOVE ASIDE")
	hs.keys("Esc")
	hs.contains("office", "!:1 error")
}

// What the user did, and what hostu did by itself, is in the App Log (tdp
// F5, dialog/note).
func TestAppLog(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 120, 40)
	hs.keys("n", "lab", "Enter", "r", "Backspace", "Backspace", "Backspace", "work", "Enter", "c", "Enter")
	hs.keys("Tab", "n", "Enter", "nas.lan", "Enter", "j", "Enter", "10.0.0.9", "Enter", "Tab", "Enter")
	hs.keys("Enter", "Tab", "Enter", "D", "Enter")
	hs.keys("1", "gg", "p", "Esc", "Enter", "Enter", "!")
	hs.contains("Made the profile lab", "Renamed the profile lab to work", "Copied the profile work to work copy",
		"Added nas.lan to the profile work copy", "Saved nas.lan in the profile work copy",
		"Deleted nas.lan from the profile work copy",
		"Kept "+hs.env.Hosts+" as it was in "+filepath.Join(hs.env.Data, original), "Wrote default to "+hs.env.Hosts,
		"Previewed the profile default")
}

// What is cancelled was not done and is not in the App Log (tdp note.7).
func TestAppLogLeavesOutCancelled(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 120, 40)
	hs.keys("G", "D", "Esc", "r", "Esc", "Tab", "n", "Esc", "D", "Esc", "1", "Enter", "Esc", "!")
	hs.lacks("Deleted", "Renamed", "Added", "Wrote")
}

func TestQuit(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("q")
	if !hs.quit {
		t.Fatal("q did not quit")
	}
}

// sudo that wants a password asks in a terminal popup; once the copy has
// run, the profile is the active one and the temporary file is gone. sh
// stands in for sudo here.
func TestSudoAsksInATerminal(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	tmp, _ := os.CreateTemp(t.TempDir(), "copy")
	tmp.Close()
	home := hs.h.store.Get("home")
	hs.send(written{p: home, elevated: &hosts.Elevated{Command: []string{"sh", "-c", "sleep 0.1"}, Temp: tmp.Name()}})
	for i := 0; i < 20 && hs.h.store.Active != "home"; i++ {
		hs.send(nil)
	}
	if hs.h.store.Active != "home" {
		t.Fatalf("the active profile is %s:\n%s", hs.h.store.Active, hs.text())
	}
	if _, err := os.Stat(tmp.Name()); err == nil {
		t.Fatal("the temporary file is left")
	}
	hs.contains("Wrote home to " + hs.env.Hosts)
}

// A long name is cut and the short columns keep their width (layout/panel):
// the count of domains stays whole.
func TestLongNameIsCut(t *testing.T) {
	t.Parallel()
	hs := start(t, sandbox(t, "active: default\nprofiles:\n  - name: default\n  - name: the client site in the north building\n    domains:\n      - domain: erp.client.lan\n        addresses: [172.16.8.10]\n"), 90, 24)
	hs.contains("NAME", "DOMAINS", "the client")
	hs.lacks("north building")
	for _, l := range hs.frame() {
		if strings.Contains(l, "the client") && (!strings.Contains(l, "…") || !strings.Contains(l, " 1 ")) {
			t.Fatalf("the name is not cut, or the count is not whole: %q", l)
		}
	}
	hs.fullWidth("a long name")
}

// The active mark is a column of its own: NAME stands over the names, the
// active one's and the others' alike, whatever an icon's width (tdp panel.5).
func TestNameStandsOverTheNames(t *testing.T) {
	eachIconWidth(t, []int{60, 120}, func(w int) string { return fmt.Sprintf("%d wide", w) }, func(t *testing.T, w int) {
		hs := newHarness(t, w, 24)
		col := func(s string) int {
			for _, l := range hs.frame() {
				if i := strings.Index(l, s); i >= 0 && strings.HasPrefix(l, "┃") {
					return terminu.Width(l[:i])
				}
			}
			t.Fatalf("no %q in [1]:\n%s", s, hs.text())
			return 0
		}
		if n, a, o := col("NAME"), col("default"), col("office"); a != n || o != n {
			t.Fatalf("NAME at %d, default (active) at %d, office at %d:\n%s", n, a, o, hs.text())
		}
	})
}

// A save that fails keeps the input open with its value, the error popup
// over it, so Enter tries again once it can (tdp popup, note).
func TestSaveFailsKeepsTheInput(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	if err := os.Chmod(hs.env.Data, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(hs.env.Data, 0o755)
	if f, err := os.CreateTemp(hs.env.Data, "w"); err == nil {
		f.Close()
		t.Skip("running as root: every directory is writable")
	}
	hs.keys("n", "lab", "Enter")
	if hs.h.store.Get("lab") != nil {
		t.Fatal("a profile was added that could not be saved")
	}
	hs.contains("FAILED")
	hs.keys("Esc")
	hs.contains("NEW PROFILE", "lab")
	os.Chmod(hs.env.Data, 0o755)
	hs.keys("Enter")
	if hs.h.store.Get("lab") == nil {
		t.Fatal("Enter did not try again")
	}
}

// paste is text pasted into hostu.
func paste(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Paste: true} }

// A name's spaces at either end go when it is confirmed; a control character
// pasted in stops it, checked before the spaces go (tdp text, input).
func TestNameSpacesAndControlCharacters(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("n", "  lab  ", "Enter")
	if hs.h.store.Get("lab") == nil {
		t.Fatalf("profiles %v", hs.h.store.Profiles)
	}
	hs.keys("n")
	hs.send(paste("work\t"))
	hs.keys("Enter")
	hs.contains("Name can't have control characters")
	if hs.h.store.Get("work") != nil {
		t.Fatal("a name with a Tab was taken")
	}
}

// Addresses go between spaces or commas; a Tab pasted in is a control
// character, stopped (tdp input).
func TestPastedTabInAddresses(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("Tab", "n", "j", "Enter")
	hs.send(paste("10.0.0.1\t10.0.0.2"))
	hs.keys("Enter")
	hs.contains("Addresses can't have control characters")
}

// A write that fails shows the line of the tool's output that says why; all
// it said is in the App Log (components, control characters).
func TestWriteFailsSaysWhy(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 120, 40)
	hs.send(written{p: hs.h.store.Get("home"), err: &hosts.ToolError{Tool: "gsudo",
		Reason: "Access is denied.", Output: "Access is denied. At line:1 char:1"}})
	hs.contains("WRITE FAILED", "Could not write it with gsudo: Access is denied.")
	hs.lacks("At line:1")
	hs.keys("Esc", "!")
	hs.contains("Output of gsudo: Access is denied. At line:1 char:1")
}

// An empty list's hint keeps only the keys for the whole panel: no Enter, no
// key for an item (tdp panel.1, M6). With no profile, [2] has none at all.
func TestEmptyListsHint(t *testing.T) {
	t.Parallel()
	hs := start(t, sandbox(t, "profiles: []\n"), 120, 40)
	hs.contains("No profiles yet", "No profile to show", "n:new")
	hs.lacks("Enter:activate", "p:preview", "Enter:edit")
	if n := strings.Count(hs.text(), "n:new"); n != 1 {
		t.Fatalf("n:new is there %d times:\n%s", n, hs.text())
	}
	hs.keys("n", "lab", "Enter")
	hs.contains("No domains in lab", "Enter:activate p:preview n:new")
	if n := strings.Count(hs.text(), "n:new"); n != 2 {
		t.Fatalf("[2] of an empty profile has no n:new:\n%s", hs.text())
	}
	hs.lacks("Enter:edit")
}

// Deleting from the Space menu leaves nothing for the menu to be about: it
// closes once the confirm is accepted (tdp T1).
func TestDeleteFromTheMenuClosesIt(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("j", "Space", "D")
	hs.contains("DELETE: office")
	hs.keys("Enter")
	hs.lacks("ITEM OPERATION", "DELETE")
	hs.keys("G", "Tab", "j", "Space", "D")
	hs.contains("DELETE: nas.home")
	hs.keys("Enter")
	hs.lacks("ITEM OPERATION", "DELETE")
}
