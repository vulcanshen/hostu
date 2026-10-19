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
}

// TestL4EveryLineIsTheTerminalsWidth is tdp L4: on every screen and under
// every popup, at many sizes from the least hostu needs up. hostu measures
// its whole screen; each popup's own frame is the lib's to measure, the lib
// draws it (tdp E8).
func TestL4EveryLineIsTheTerminalsWidth(t *testing.T) {
	var cases []sized
	for _, sz := range [][2]int{{80, 24}, {80, 40}, {100, 30}, {120, 40}, {160, 50}} {
		for _, st := range states {
			cases = append(cases, sized{w: sz[0], h: sz[1], name: st.name, keys: st.keys})
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
		for _, small := range [][2]int{{80, 40}, {120, 24}, {80, 24}, {80, 5}, {1, 1}} {
			cases = append(cases, sized{w: 120, h: 40, name: st.name, keys: st.keys, small: small})
		}
	}
	eachIconWidth(t, cases, func(c sized) string { return fmt.Sprintf("%s, then %dx%d", c.name, c.small[0], c.small[1]) }, checkSized)
}

func checkSized(t *testing.T, c sized) {
	hs := newHarness(t, c.w, c.h)
	hs.keys(c.keys...)
	if c.small[0] > 0 {
		hs.send(tea.WindowSizeMsg{Width: c.small[0], Height: c.small[1]})
	}
	hs.fullWidth(c.name)
}

// TestL1BelowTheLeastSize is tdp L1: smaller than 80×24 either way, only
// hostu's name shows, and under it the size it needs when that fits. No key
// acts but Ctrl-C twice, and all comes back as the terminal grows.
func TestL1BelowTheLeastSize(t *testing.T) {
	const hint = "needs a terminal of at least 80×24"
	for _, sz := range [][2]int{{79, 24}, {80, 23}, {40, 12}, {30, 5}, {80, 1}, {1, 1}} {
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
			if sz[0] >= terminu.Width(hint) && sz[1] >= 2 {
				hs.contains(hint)
			} else {
				hs.lacks(hint)
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

func TestActivateWritesTheHostsFile(t *testing.T) {
	t.Parallel()
	hs := newHarness(t, 80, 40)
	hs.keys("G", "Enter") // home
	hs.contains("ACTIVATE: home", "The hosts file becomes the 2 domains of home.", "Write home to "+hs.env.Hosts+"?")
	hs.lacks("has no localhost")
	hs.keys("Enter")
	home := hs.h.store.Get("home")
	if hs.hostsFile() != hosts.Format("home", home.Domains) {
		t.Fatalf("the hosts file is\n%s", hs.hostsFile())
	}
	hs.contains("home is in "+hs.env.Hosts+" now", "active: home")
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
	hs.lacks("hosts file differs")
	hs.keys("Tab", "j", "D", "Enter") // broadcasthost out of default
	hs.contains("hosts file differs")
	if hs.hostsFile() != systemHosts {
		t.Fatal("a change to the profile wrote the hosts file")
	}
	hs.keys("1", "Enter", "Enter")
	hs.lacks("hosts file differs")
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
	hs.contains("Imported 3 domains from " + hs.env.Hosts + ", skipped 1 line")
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
	hs.contains("DELETE: office", "Its 1 domain go with it.", "Delete the profile office?")
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

// A profiles.yaml hostu can't read is left as it is: it says so, and every
// change after it fails instead of writing over the file.
func TestBrokenProfilesAreLeftAlone(t *testing.T) {
	t.Parallel()
	const broken = "profiles: [\n"
	hs := start(t, sandbox(t, broken), 80, 40)
	hs.contains("CAN'T READ")
	hs.keys("Esc", "n", "lab", "Enter")
	hs.contains("could not be read, so hostu")
	if hs.saved() != broken {
		t.Fatalf("profiles.yaml is now %q", hs.saved())
	}
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
	hs.contains("home is in " + hs.env.Hosts + " now")
}

// A long name is cut and the short columns keep their width (layout/panel):
// the count of domains stays whole.
func TestLongNameIsCut(t *testing.T) {
	t.Parallel()
	hs := start(t, sandbox(t, "active: default\nprofiles:\n  - name: default\n  - name: the client site in the north building\n    domains:\n      - domain: erp.client.lan\n        addresses: [172.16.8.10]\n"), 80, 24)
	hs.contains("NAME", "DOMAINS", "the client…")
	hs.lacks("north building")
	for _, l := range hs.frame() {
		if strings.Contains(l, "the client…") && !strings.Contains(l, " 1 ") {
			t.Fatalf("the count is not whole: %q", l)
		}
	}
	hs.fullWidth("a long name")
}
