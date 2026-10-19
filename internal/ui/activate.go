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

// written is a write of the hosts file that has ended: elevated when it waits
// for a password.
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

// activate writes p to the hosts file, off the update loop: sudo may take a
// moment, and Windows' UAC waits for the user.
func (h *hostu) activate(p *profile.Profile) tea.Cmd {
	text := hosts.Format(p.Name, p.Domains)
	path, data := h.env.Hosts, h.env.Data
	return func() tea.Msg {
		if err := keepOriginal(path, data); err != nil {
			return written{p: p, err: err}
		}
		e, err := hosts.Write(path, text)
		return written{p: p, elevated: e, err: err}
	}
}

// keepOriginal copies the hosts file into the data directory the first time
// hostu writes it: the profiles leave out its comments.
func keepOriginal(path, data string) error {
	keep := filepath.Join(data, original)
	if _, err := os.Stat(keep); !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := os.MkdirAll(data, 0o755); err != nil {
		return err
	}
	return os.WriteFile(keep, b, 0o644)
}

// onMsg takes what the lib leaves: the end of a write.
func (h *hostu) onMsg(msg tea.Msg) tea.Cmd {
	w, ok := msg.(written)
	if !ok {
		return nil
	}
	switch {
	case w.err != nil:
		return h.a.ErrorAbout("Write failed", h.env.Hosts, sentence(w.err))
	case w.elevated != nil:
		// sudo wants a password: it asks in a terminal popup, which closes
		// when the copy ends (components/dialog/terminal).
		e := w.elevated
		return h.a.OpenTerminal(&terminu.Terminal{Title: "Sudo", Subject: h.env.Hosts, Command: e.Command, Ends: true,
			OnExit: func(err error) tea.Cmd {
				e.Done()
				if err != nil {
					return h.a.ErrorAbout("Write failed", h.env.Hosts, e.Command[0]+" did not write it: "+sentence(err))
				}
				return h.wrote(w.p)
			}})
	}
	return h.wrote(w.p)
}

// wrote makes p the active profile once the hosts file has it.
func (h *hostu) wrote(p *profile.Profile) tea.Cmd {
	h.readHosts()
	active := h.store.Active
	h.store.Active = p.Name
	err := h.save()
	if err != nil {
		h.store.Active = active
	}
	h.refresh()
	if err != nil {
		return h.a.ErrorAbout("Can't save", h.store.Path(), "The hosts file has "+p.Name+", but "+sentence(err))
	}
	return h.a.Toast(terminu.Success, p.Name+" is in "+h.env.Hosts+" now")
}
