package ui

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/hostu/internal/hosts"
	"github.com/vulcanshen/hostu/internal/profile"
)

// original is the hosts file as it was before hostu first wrote it, in the
// data directory (tdp E3: lost for good when deleted).
const original = "hosts.original"

// written is a write of the hosts file that has ended: elevated when the copy
// is left to run as the administrator.
type written struct {
	p        *profile.Profile
	elevated *hosts.Elevated
	err      error
}

// activateAsk is the question before p is written: how much it writes, and a
// warning when it has no localhost.
func (h *hostu) activateAsk(p *profile.Profile) *terminu.Ask {
	ask := &terminu.Ask{Title: "Activate", Subject: p.Name, Verb: "write",
		Details:  []string{fmt.Sprintf("The hosts file gets %s, and nothing else.", count(len(p.Domains), "domain"))},
		Question: "Write " + p.Name + " to " + h.env.Hosts + "?"}
	if !hosts.Has(p.Domains, "localhost") {
		ask.Details = append(ask.Details, p.Name+" has no localhost: programs that connect to localhost stop working.")
		ask.Warn = []int{1}
	}
	return ask
}

// activate writes p to the hosts file, off the update loop: Windows' UAC
// waits for the user.
func (h *hostu) activate(p *profile.Profile) tea.Cmd {
	text := hosts.Format(p.Name, p.Domains)
	path, data := h.env.Hosts, h.env.Data
	return func() tea.Msg {
		kept, err := keepOriginal(path, data)
		if err != nil {
			return written{p: p, err: err}
		}
		if kept != "" {
			h.a.Log(terminu.Info, "Kept "+short(path)+" as it was in "+short(kept))
		}
		e, err := hosts.Write(path, text)
		return written{p: p, elevated: e, err: err}
	}
}

// keepOriginal copies the hosts file into the data directory the first time
// hostu writes it: the profiles leave out its comments. kept is the copy it
// made, "" when there was one already or no hosts file to copy.
func keepOriginal(path, data string) (kept string, err error) {
	keep := filepath.Join(data, original)
	if _, err := os.Stat(keep); !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(data, 0o755); err != nil {
		return "", err
	}
	return keep, os.WriteFile(keep, b, 0o644)
}

// onMsg takes what the lib leaves: the end of a write.
func (h *hostu) onMsg(msg tea.Msg) tea.Cmd {
	w, ok := msg.(written)
	if !ok {
		return nil
	}
	switch {
	case w.err != nil:
		return h.writeFailed(w.err)
	case w.elevated != nil:
		// The lib copies it as the administrator: at once when sudo needs no
		// password; otherwise it asks for it in a password popup, and hostu
		// never sees it. doas asks in a terminal popup (tdp terminal.7).
		e := w.elevated
		tool := terminu.Sudo
		if e.Tool == "doas" {
			tool = terminu.Doas
		}
		return h.a.Elevate(terminu.Elevate{Tool: tool, Command: e.Command, Verb: "write", Subject: h.env.Hosts,
			Done: func(err error) tea.Cmd {
				e.Done()
				var ee *terminu.ElevateError
				switch {
				case errors.As(err, &ee):
					return h.writeFailed(hosts.Failed(e.Tool, ee.Lines, ee.Err))
				case err != nil:
					return h.writeFailed(err)
				}
				return h.wrote(w.p)
			},
			OnCancel: func() tea.Cmd { e.Done(); return nil }})
	}
	return h.wrote(w.p)
}

// writeFailed says why the hosts file was not written.
func (h *hostu) writeFailed(err error) tea.Cmd {
	cmd := h.a.ErrorAbout("Write failed", h.env.Hosts, sentence(err))
	// The popup has the line that says why; all the tool said goes in the
	// App Log (components, control characters).
	var te *hosts.ToolError
	if errors.As(err, &te) && te.Output != te.Reason {
		h.a.Log(terminu.Info, "Output of "+te.Tool+": "+te.Output)
	}
	return cmd
}

// wrote makes p the active profile once the hosts file has it.
func (h *hostu) wrote(p *profile.Profile) tea.Cmd {
	h.readHosts()
	active := h.store.Active
	h.store.Active = p.Name
	err := h.store.Save()
	if err != nil {
		h.store.Active = active
	}
	h.refresh()
	if err != nil {
		return h.a.ErrorAbout("Can't save", h.store.Path(), "The hosts file has "+p.Name+", but it is not kept as the active profile. "+sentence(err))
	}
	// The verb first: the App Log takes the toast as it is, a capital first
	// (tdp note-78), and the name stays as it is.
	return h.a.Toast(terminu.Success, "Wrote "+p.Name+" to "+h.env.Hosts)
}
