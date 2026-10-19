package ui

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/hostu/internal/hosts"
	"github.com/vulcanshen/hostu/internal/profile"
)

// profileActions are [1] PROFILES': Enter writes the profile to the hosts
// file; the rest make, change and take away profiles.
func (h *hostu) profileActions() []terminu.Action {
	return []terminu.Action{
		{Key: "Enter", Label: "Activate", Desc: "write it to the hosts file", Scope: terminu.ItemScope, Hint: "activate",
			AskFor: func() *terminu.Ask { return h.activateAsk(h.current()) },
			Do:     func() tea.Cmd { return h.activate(h.current()) }},
		{Key: "p", Label: "preview", Desc: "the hosts file it writes", Scope: terminu.ItemScope, Hint: "preview",
			Do: func() tea.Cmd { return h.preview(h.current()) }},
		{Key: "r", Label: "rename", Desc: "give it another name", Scope: terminu.ItemScope,
			Do: func() tea.Cmd { return h.rename(h.current()) }},
		{Key: "c", Label: "copy", Desc: "a new profile with its domains", Scope: terminu.ItemScope,
			Do: func() tea.Cmd { return h.copyProfile(h.current()) }},
		// The active profile is in the hosts file: another is activated
		// before it goes (tdp M6).
		{Key: "D", Label: "Delete", Desc: "take it away, with its domains", Scope: terminu.ItemScope,
			Disabled: func() bool { p := h.current(); return p != nil && p.Name == h.store.Active },
			AskFor: func() *terminu.Ask {
				p := h.current()
				return &terminu.Ask{Title: "Delete", Subject: p.Name, Verb: "delete",
					Details:  []string{fmt.Sprintf("Its %s go with it.", count(len(p.Domains), "domain"))},
					Question: "Delete the profile " + p.Name + "?"}
			},
			Do: func() tea.Cmd { return h.deleteProfile(h.current()) }},
		{Key: "n", Label: "new", Desc: "an empty profile", Scope: terminu.PanelScope, Hint: "new", Do: h.newProfile},
		{Key: "i", Label: "import", Desc: "the hosts file as it is now, as a new profile", Scope: terminu.PanelScope,
			Do: h.importProfile},
	}
}

// nameInput is a text input for a profile's name: not empty, no space at
// either end, no other profile's (but's own allowed).
func (h *hostu) nameInput(in terminu.TextInput, but *profile.Profile) terminu.TextInput {
	in.Name = "Name"
	in.Check = func(s string) error {
		switch {
		case s == "":
			return terminu.ErrEmpty
		case s != strings.TrimSpace(s):
			return errors.New("A name has no space at either end")
		case h.store.Taken(s, but):
			return fmt.Errorf("There is a profile named %s already", s)
		}
		return nil
	}
	return in
}

// add puts p after the others, saved, the cursor on it (tdp K3).
func (h *hostu) add(p *profile.Profile) tea.Cmd {
	h.store.Profiles = append(h.store.Profiles, p)
	if err := h.save(); err != nil {
		h.store.Remove(p)
		return h.a.ErrorAbout("Can't save", h.store.Path(), sentence(err))
	}
	h.refresh()
	h.profiles.Select(p.Name)
	h.showDomains()
	return nil
}

func (h *hostu) newProfile() tea.Cmd {
	return h.a.Text(h.nameInput(terminu.TextInput{Title: "New profile", Glyph: glyphPencil,
		Prompt: "Name of the new profile", Verb: "add",
		Done: func(s string) tea.Cmd { return h.add(&profile.Profile{Name: s}) }}, nil))
}

func (h *hostu) rename(p *profile.Profile) tea.Cmd {
	return h.a.Text(h.nameInput(terminu.TextInput{Title: "Rename", Subject: p.Name, Glyph: glyphPencil,
		Prompt: "New name for " + p.Name, Old: p.Name, Verb: "rename",
		Done: func(s string) tea.Cmd {
			old, active := p.Name, h.store.Active
			p.Name = s
			if active == old {
				h.store.Active = s
			}
			if err := h.save(); err != nil {
				p.Name, h.store.Active = old, active
				return h.a.ErrorAbout("Can't save", h.store.Path(), sentence(err))
			}
			h.refresh()
			h.profiles.Select(s)
			h.showDomains()
			return nil
		}}, p))
}

func (h *hostu) copyProfile(p *profile.Profile) tea.Cmd {
	name := p.Name + " copy"
	for i := 2; h.store.Taken(name, nil); i++ {
		name = fmt.Sprintf("%s copy %d", p.Name, i)
	}
	return h.a.Text(h.nameInput(terminu.TextInput{Title: "Copy", Subject: p.Name, Glyph: glyphCopy,
		Prompt: "Name of the copy", Suggest: name, Verb: "copy",
		Done: func(s string) tea.Cmd {
			domains := make([]hosts.Domain, len(p.Domains))
			for i, d := range p.Domains {
				domains[i] = hosts.Domain{Name: d.Name, Addresses: slices.Clone(d.Addresses)}
			}
			return h.add(&profile.Profile{Name: s, Domains: domains})
		}}, nil))
}

// importProfile reads the hosts file again and keeps what is in it as a new
// profile.
func (h *hostu) importProfile() tea.Cmd {
	return h.a.Text(h.nameInput(terminu.TextInput{Title: "Import", Subject: h.env.Hosts, Glyph: glyphImport,
		Prompt: "Name of the profile made from it", Suggest: "imported", Verb: "import",
		Done: func(s string) tea.Cmd {
			b, err := os.ReadFile(h.env.Hosts)
			if err != nil {
				return h.a.ErrorAbout("Can't read", h.env.Hosts, sentence(err))
			}
			domains, skipped := hosts.Parse(string(b))
			if cmd := h.add(&profile.Profile{Name: s, Domains: domains}); cmd != nil {
				return cmd
			}
			say := fmt.Sprintf("Imported %s from %s", count(len(domains), "domain"), h.env.Hosts)
			if skipped > 0 {
				say += ", skipped " + count(skipped, "line")
			}
			return h.a.Toast(terminu.Success, say)
		}}, nil))
}

func (h *hostu) deleteProfile(p *profile.Profile) tea.Cmd {
	i := slices.Index(h.store.Profiles, p)
	h.store.Remove(p)
	if err := h.save(); err != nil {
		h.store.Profiles = slices.Insert(h.store.Profiles, i, p)
		return h.a.ErrorAbout("Can't save", h.store.Path(), sentence(err))
	}
	h.refresh()
	return nil
}

// preview shows the hosts file p writes, as it writes it.
func (h *hostu) preview(p *profile.Profile) tea.Cmd {
	text := strings.ReplaceAll(hosts.Format(p.Name, p.Domains), "\r", "")
	return h.a.Viewer(&terminu.Viewer{Title: "Hosts file", Subject: p.Name, Glyph: glyphHosts,
		Lines: strings.Split(strings.TrimSuffix(text, "\n"), "\n")})
}

// count is n things, in the plural but for one.
func count(n int, thing string) string {
	if n == 1 {
		return "1 " + thing
	}
	return fmt.Sprintf("%d %ss", n, thing)
}
