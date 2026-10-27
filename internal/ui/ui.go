// Package ui is hostu's screen: the profiles on the left, the domains of the
// one under the cursor on the right, the hosts file and the active profile
// on the statusbar.
package ui

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
	"gopkg.in/yaml.v3"

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
	// first is that there is no profiles.yaml yet: begin makes it.
	first bool
	// onDisk is the hosts file as hostu last read it; diskErr why it could
	// not be read.
	onDisk  []hosts.Domain
	diskErr error

	profiles, domains *terminu.List
	pp, dp            *terminu.Panel
	unread            []unread // to ask about once the program runs
}

// unread is a file of hostu's own it could not read, and fresh what hostu
// does once it is moved aside: start as the first time, with none (tdp E3,
// when it cannot be read).
type unread struct {
	path  string
	err   error
	fresh func() tea.Cmd
}

// New makes hostu with its things where env says.
func New(env Env) *terminu.App { return newHostu(env).a }

func newHostu(env Env) *hostu {
	h := &hostu{a: terminu.New("hostu"), env: env}
	// The least size (tdp L1): two panels side by side and every popup fit
	// in 60 columns, and in 24 rows, the height a new terminal window often
	// has.
	h.a.MinWidth, h.a.MinHeight = 60, 24
	h.a.QuitDesc = "leave hostu"
	h.a.Splash = splash()
	h.a.Status = h.status
	h.a.OnMsg = h.onMsg
	h.a.Start = h.askUnread
	// The icon's width is known once the program runs: the names line up
	// again with the right space for it.
	h.a.OnResize = func(int, int) tea.Cmd { h.refresh(); return nil }

	cfg, err := config.Load(env.Config)
	if err != nil {
		// Moved aside there is none, as the first time: hostu has the
		// defaults already and writes no config.yaml.
		h.unread = append(h.unread, unread{config.Path(env.Config), err, func() tea.Cmd { return nil }})
	}
	h.a.InputTabWidth = cfg.InputTabWidth
	h.load()

	// The active mark is a column of its own, with no title: NAME stands
	// over the names (tdp panel.5).
	h.profiles = &terminu.List{
		Columns:   []terminu.Column{{}, {Title: "Name", Shrink: true}, {Title: "Domains", Number: true, Drop: 1}},
		EmptyFact: "No profiles yet",
		EmptyHint: "Press [n] to make one, [i] to read the hosts file into one, or [Space] to see what you can do here",
		OnMove:    func(terminu.Row) tea.Cmd { h.showDomains(); return nil },
	}
	h.domains = &terminu.List{Columns: []terminu.Column{{Title: "Domain", Shrink: true}, {Title: "Addresses", Shrink: true, Drop: 1}}}
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

// load reads profiles.yaml. One that can't be read is asked about (tdp E3),
// and once it is moved aside there is none: the first start.
func (h *hostu) load() {
	s, found, err := profile.Load(h.env.Data)
	h.store, h.first = s, !found
	if err != nil {
		h.unread = append(h.unread, unread{s.Path(), err, func() tea.Cmd { h.load(); return nil }})
	}
}

// begin starts hostu once its own files are read, nothing written before:
// the hosts file is read as it is now. The first time, with no
// profiles.yaml, it becomes the profile default, the active one; the lines
// of it that were skipped are told in a Warning toast.
func (h *hostu) begin() tea.Cmd {
	skipped := h.readHosts()
	defer h.refresh()
	switch {
	case h.diskErr != nil:
		return h.a.ErrorAbout("Can't read", h.env.Hosts, sentence(h.diskErr))
	case !h.first:
		return nil
	}
	s := h.store
	s.Profiles = []*profile.Profile{{Name: "default", Domains: h.onDisk}}
	s.Active = "default"
	if err := s.Save(); err != nil {
		return h.a.ErrorAbout("Can't save", s.Path(), sentence(err))
	}
	say := "Made the profile default from " + h.env.Hosts
	if skipped > 0 {
		return h.a.Toast(terminu.Warning, say+", skipped "+count(skipped, "line"))
	}
	return h.a.Toast(terminu.Info, say)
}

// askUnread asks about the first file of hostu's own it could not read, and
// about the next once that one is moved aside; Esc leaves hostu (tdp E3).
// With none left, hostu begins.
func (h *hostu) askUnread() tea.Cmd {
	if len(h.unread) == 0 {
		return h.begin()
	}
	u := h.unread[0]
	h.unread = h.unread[1:]
	return h.a.MoveAside(u.path, reason(u.err), func() tea.Cmd { return tea.Batch(u.fresh(), h.askUnread()) })
}

// reason is err as MOVE ASIDE gives it, after the path: without the path
// again, without yaml's "yaml: ", on one line, and a value of the wrong
// kind in the user's words, not Go's.
func reason(err error) error {
	var pe *os.PathError
	var te *yaml.TypeError
	switch {
	case errors.As(err, &pe):
		return pe.Err
	case errors.As(err, &te):
		why := make([]string, len(te.Errors))
		for i, e := range te.Errors {
			why[i] = wrongKind(e)
		}
		return errors.New(strings.Join(why, "; "))
	}
	if s, ok := strings.CutPrefix(err.Error(), "yaml: "); ok {
		return errors.New(s)
	}
	return err
}

// unmarshal is one of yaml's errors for a value of the wrong kind:
// "line 1: cannot unmarshal !!int `3` into []*profile.Profile".
var unmarshal = regexp.MustCompile("^(line \\d+): cannot unmarshal !!(\\w+)(?: `(.*)`)? into (.+)$")

// wrongKind is e, one of yaml's errors, as "line 1: 3 is not a list".
func wrongKind(e string) string {
	m := unmarshal.FindStringSubmatch(e)
	if m == nil {
		return e
	}
	what := m[3]
	if what == "" {
		what = map[string]string{"map": "a mapping", "seq": "a list"}[m[2]]
	}
	want := "a mapping"
	switch {
	case strings.HasPrefix(m[4], "[]"):
		want = "a list"
	case m[4] == "string":
		want = "text"
	case m[4] == "int":
		want = "a number"
	}
	if what == "" {
		return m[1] + ": not " + want
	}
	return m[1] + ": " + what + " is not " + want
}

// short is p with the home directory written ~, as terminu-go writes paths
// in the App Log.
func short(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if rest, ok := strings.CutPrefix(p, home+string(filepath.Separator)); ok {
		return "~" + string(filepath.Separator) + rest
	}
	return p
}

// readHosts reads the hosts file as it is now; skipped is how many of its
// lines were not an address and names.
func (h *hostu) readHosts() (skipped int) {
	b, err := os.ReadFile(h.env.Hosts)
	h.diskErr = err
	h.onDisk = nil
	if err == nil {
		h.onDisk, skipped = hosts.Parse(string(b))
	}
	return skipped
}

// refresh shows the profiles as they are now.
func (h *hostu) refresh() {
	rows := make([]terminu.Row, len(h.store.Profiles))
	// The mark's column is as wide with no profile active: the names keep
	// their place (tdp L2).
	blank := strings.Repeat(" ", terminu.Width(glyphActive))
	for i, p := range h.store.Profiles {
		rows[i] = terminu.Row{ID: p.Name, Cells: []string{blank, p.Name, fmt.Sprint(len(p.Domains))}}
		if p.Name == h.store.Active {
			rows[i].Cells[0] = glyphActive
			rows[i].Styled = [][]terminu.Span{{{Text: glyphActive, Style: terminu.Fg(terminu.Green)}}}
		}
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

// status is the statusbar: the hosts file and whether it has the active
// profile, then the active profile. Nothing moves as they change (tdp L2):
// the path is at most a third of the width, cut from the front, and the
// state after it is padded to its widest; the profile's name is last.
func (h *hostu) status(w int) []terminu.Span {
	grey := terminu.Fg(terminu.Overlay0)
	path := h.env.Hosts
	if room := max(12, w/3); terminu.Width(path) > room {
		path = terminu.TruncateLeft(path, room)
	}
	state, colour := "", terminu.Overlay0
	switch {
	case h.diskErr != nil:
		state, colour = "(can't read)", terminu.Red
	case h.differs():
		state, colour = "(differs)", terminu.Peach
	case h.store.Get(h.store.Active) != nil:
		state = "(in sync)"
	}
	spans := []terminu.Span{{Text: "hosts: " + path + " ", Style: grey},
		{Text: terminu.Pad(state, len("(can't read)")), Style: terminu.Fg(colour)}, {Text: "  active: ", Style: grey}}
	if h.store.Active == "" {
		return append(spans, terminu.Span{Text: "none", Style: grey})
	}
	return append(spans, terminu.Span{Text: h.store.Active, Style: terminu.Fg(terminu.Lavender)})
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
