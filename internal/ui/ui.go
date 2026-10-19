// Package ui is hostu's screen: the profiles on the left, the domains of the
// one under the cursor on the right, the hosts file and the active profile
// on the statusbar.
package ui

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/hostu/internal/config"
	"github.com/vulcanshen/hostu/internal/hosts"
	"github.com/vulcanshen/hostu/internal/profile"
	"github.com/vulcanshen/hostu/internal/version"
)

// icon is hostu's family mark (tdp S5), drawn by the splash: a copy of
// docs/icon.svg, which go:embed can't reach from here (a test holds the two
// the same, tdp E6).
//
//go:embed icon.svg
var icon []byte

// The glyphs hostu draws (the app's choice, tdp P5).
const (
	glyphActive = "\U000F05E0" // nf-md-check_circle: the profile in the hosts file
	glyphPencil = "\U000F03EB" // nf-md-pencil: a name to type
	glyphCopy   = "\U000F018F" // nf-md-content_copy
	glyphImport = "\U000F02FA" // nf-md-import
	glyphDomain = "\U000F059F" // nf-md-web
	glyphHosts  = "\U000F0219" // nf-md-file_document
)

// Env is where hostu keeps its things and the hosts file it writes.
type Env struct {
	Config, Data, Hosts string
}

// hostu is the app and what it shows.
type hostu struct {
	a     *terminu.App
	env   Env
	store *profile.Store
	// storeErr is why profiles.yaml could not be read: hostu then writes
	// nothing over it.
	storeErr error
	// onDisk is the hosts file as hostu last read it; diskErr why it could
	// not be read.
	onDisk  []hosts.Domain
	diskErr error

	profiles, domains *terminu.List
	pp, dp            *terminu.Panel
	start             []func() tea.Cmd // what to say once the program runs
}

// New makes hostu with its things where env says.
func New(env Env) *terminu.App { return newHostu(env).a }

func newHostu(env Env) *hostu {
	h := &hostu{a: terminu.New("hostu"), env: env}
	// The least size (tdp L1): two panels and every popup fit in 80×24, the
	// size a new terminal window often has.
	h.a.MinWidth, h.a.MinHeight = 80, 24
	h.a.QuitDesc = "leave hostu"
	h.a.Splash = splash()
	h.a.Status = h.status
	h.a.OnMsg = h.onMsg
	h.a.Start = func() tea.Cmd {
		h.refresh()
		var cmds []tea.Cmd
		for _, say := range h.start {
			cmds = append(cmds, say())
		}
		return tea.Batch(cmds...)
	}
	// The icon's width is known once the program runs: the names line up
	// again with the right space for it.
	h.a.OnResize = func(int, int) tea.Cmd { h.refresh(); return nil }

	cfg, err := config.Load(env.Config)
	if err != nil {
		h.fail("Can't read", env.Config+"/"+config.File, err)
	}
	h.a.InputTabWidth = cfg.InputTabWidth
	h.readHosts()
	if h.diskErr != nil {
		h.fail("Can't read", env.Hosts, h.diskErr)
	}
	h.load()

	h.profiles = &terminu.List{
		Columns:   []terminu.Column{{Title: "Name"}, {Title: "Domains", Number: true, Drop: 1}},
		EmptyFact: "No profiles yet",
		EmptyHint: "Press [n] to make one, [i] to read the hosts file into one, or [Space] to see what you can do here",
		OnMove:    func(terminu.Row) tea.Cmd { h.showDomains(); return nil },
	}
	h.domains = &terminu.List{Columns: []terminu.Column{{Title: "Domain"}, {Title: "Addresses", Drop: 1}}}
	h.dp = &terminu.Panel{Label: "Domains", Body: h.domains, Actions: h.domainActions()}
	h.pp = &terminu.Panel{Label: "Profiles", Body: h.profiles, Actions: h.profileActions(), Details: h.dp}
	h.a.AddScreen(&terminu.Screen{Name: "Profiles", Panels: []*terminu.Panel{h.pp, h.dp},
		Layout: func(w, ht int) []terminu.Rect {
			left := w / 3
			return []terminu.Rect{{X: 0, Y: 0, W: left, H: ht}, {X: left, Y: 0, W: w - left, H: ht}}
		}})
	h.refresh()
	return h
}

// splash is the family's easter egg, V on a panel (tdp S).
func splash() terminu.Splash {
	s, err := terminu.NewSplash(icon, "hostu", version.Display(), "Switch the hosts file between profiles")
	if err != nil {
		panic(err) // the embedded icon is broken: a mistake in the program
	}
	return s
}

// fail is an error popup once the program runs.
func (h *hostu) fail(title, subject string, err error) {
	h.start = append(h.start, func() tea.Cmd { return h.a.ErrorAbout(title, subject, sentence(err)) })
}

// load reads profiles.yaml. The first time there is none, the hosts file as
// it is becomes the profile default, the active one.
func (h *hostu) load() {
	s, found, err := profile.Load(h.env.Config)
	h.store = s
	switch {
	case err != nil:
		h.storeErr = err
		h.fail("Can't read", s.Path(), err)
	case !found && h.diskErr == nil:
		s.Profiles = []*profile.Profile{{Name: "default", Domains: h.onDisk}}
		s.Active = "default"
		if err := h.save(); err != nil {
			h.fail("Can't save", s.Path(), err)
			return
		}
		h.start = append(h.start, func() tea.Cmd { return h.a.Toast(terminu.Info, "Made the profile default from "+h.env.Hosts) })
	}
}

// save writes profiles.yaml, unless it could not be read.
func (h *hostu) save() error {
	if h.storeErr != nil {
		return fmt.Errorf("%s could not be read, so hostu leaves it as it is: %w", h.store.Path(), h.storeErr)
	}
	return h.store.Save()
}

// readHosts reads the hosts file as it is now.
func (h *hostu) readHosts() {
	b, err := os.ReadFile(h.env.Hosts)
	h.diskErr = err
	h.onDisk = nil
	if err == nil {
		h.onDisk, _ = hosts.Parse(string(b))
	}
}

// refresh shows the profiles as they are now.
func (h *hostu) refresh() {
	rows := make([]terminu.Row, len(h.store.Profiles))
	blank := strings.Repeat(" ", terminu.Width(glyphActive)+1)
	for i, p := range h.store.Profiles {
		name := blank + p.Name
		if p.Name == h.store.Active {
			name = terminu.Fg(terminu.Green).Render(glyphActive) + " " + p.Name
		}
		rows[i] = terminu.Row{ID: p.Name, Cells: []string{name, fmt.Sprint(len(p.Domains))}}
	}
	h.profiles.SetRows(rows)
	h.showDomains()
}

// current is the profile under [1]'s cursor, nil for none.
func (h *hostu) current() *profile.Profile {
	r, ok := h.profiles.Current()
	if !ok {
		return nil
	}
	return h.store.Get(r.ID)
}

// showDomains puts the domains of the profile under the cursor in [2], its
// name in the label (layout/panel, details).
func (h *hostu) showDomains() {
	p := h.current()
	if p == nil {
		h.dp.Name = ""
		h.domains.SetRows(nil)
		h.domains.EmptyFact, h.domains.EmptyHint = "No profile to show", "Press [1] to go to the profiles"
		return
	}
	h.dp.Name = p.Name
	rows := make([]terminu.Row, len(p.Domains))
	for i, d := range p.Domains {
		rows[i] = terminu.Row{ID: d.Name, Cells: []string{d.Name, strings.Join(d.Addresses, ", ")}}
	}
	h.domains.SetRows(rows)
	h.domains.EmptyFact = "No domains in " + p.Name
	h.domains.EmptyHint = "Press [n] to add one, or [Space] to see what you can do here"
}

// status is the statusbar: the hosts file, and the active profile with
// whether the hosts file still has it. The path takes a third of the width
// at most, cut from the front, so that what follows it is seen and never moves
// as it changes (tdp L2).
func (h *hostu) status(w int) string {
	grey := terminu.Fg(terminu.Overlay0)
	path := h.env.Hosts
	if room := max(12, w/3); terminu.Width(path) > room {
		path = terminu.TruncateLeft(path, room)
	}
	text := grey.Render("hosts: " + path + "  active: ")
	if h.store.Active == "" {
		text += grey.Render("none")
	} else {
		text += terminu.Fg(terminu.Lavender).Render(h.store.Active)
	}
	switch {
	case h.diskErr != nil:
		text += terminu.Fg(terminu.Red).Render("  can't read the hosts file")
	case h.differs():
		text += terminu.Fg(terminu.Peach).Render("  hosts file differs")
	}
	return terminu.Truncate(text, w)
}

// differs reports that the hosts file is not what the active profile writes:
// the profile changed since, or something else wrote the file.
func (h *hostu) differs() bool {
	p := h.store.Get(h.store.Active)
	return p != nil && h.diskErr == nil && !hosts.Same(p.Domains, h.onDisk)
}

// sentence is err as a sentence: a capital first and a full stop.
func sentence(err error) string {
	s := err.Error()
	var pe *os.PathError
	if errors.As(err, &pe) {
		s = pe.Err.Error() // the path is in the title
	}
	if s == "" {
		return s
	}
	s = strings.ToUpper(s[:1]) + s[1:]
	if !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s
}
