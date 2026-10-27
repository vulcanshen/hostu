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
					Details:  []string{goesWith(len(p.Domains))},
					Question: "Delete the profile " + p.Name + "?"}
			},
			Do: func() tea.Cmd { return h.deleteProfile(h.current()) }},
		{Key: "n", Label: "new", Desc: "an empty profile", Scope: terminu.PanelScope, Hint: "new", Do: h.newProfile},
		{Key: "i", Label: "import", Desc: "the hosts file as it is now, as a new profile", Scope: terminu.PanelScope,
			Do: h.importProfile},
	}
}

// goesWith says what goes with a profile of n domains when it is deleted.
func goesWith(n int) string {
	switch n {
	case 0:
		return "It has no domains."
	case 1:
		return "Its one domain goes with it."
	}
	return fmt.Sprintf("Its %d domains go with it.", n)
}

// nameInput is a text input for a profile's name: not empty, no space at
// either end, no control character, no other profile's (but's own allowed).
// Commit saves what it is for, so that a save that fails keeps the input
// open with its value, the error popup over it (tdp popup, an action failing).
func (h *hostu) nameInput(in terminu.TextInput, but *profile.Profile, commit func(string) error) terminu.TextInput {
	in.Name = "Name"
	in.Check = func(s string) error {
		switch {
		case s == "":
			return terminu.ErrEmpty
		case profile.CheckName(s) != nil:
			return profile.CheckName(s)
		case h.store.Taken(s, but):
			return fmt.Errorf("There is a profile named %s already", s)
		}
		return nil
	}
	in.Commit = func(s string) error {
		if err := commit(s); err != nil {
			return errors.New(sentence(err))
		}
		return nil
	}
	return in
}

// add puts p after the others, saved, the cursor on it (tdp K3).
func (h *hostu) add(p *profile.Profile) error {
	h.store.Profiles = append(h.store.Profiles, p)
	if err := h.store.Save(); err != nil {
		h.store.Remove(p)
		return err
	}
	h.refresh()
	h.profiles.Select(p.Name)
	h.showDomains()
	return nil
}

// unused is a name for a new profile from base, base itself when no profile
// has it yet.
func (h *hostu) unused(base string) string {
	name := base
	for i := 2; h.store.Taken(name, nil); i++ {
		name = fmt.Sprintf("%s %d", base, i)
	}
	return name
}

func (h *hostu) newProfile() tea.Cmd {
	return h.a.Text(h.nameInput(terminu.TextInput{Title: "New profile", Glyph: glyphPencil,
		Prompt: "Name of the new profile", Verb: "add"}, nil,
		func(s string) error {
			if err := h.add(&profile.Profile{Name: s}); err != nil {
				return err
			}
			h.a.Log(terminu.Success, "Made the profile "+s)
			return nil
		}))
}

func (h *hostu) rename(p *profile.Profile) tea.Cmd {
	return h.a.Text(h.nameInput(terminu.TextInput{Title: "Rename", Subject: p.Name, Glyph: glyphPencil,
		Prompt: "New name for " + p.Name, Old: p.Name, Verb: "rename"}, p,
		func(s string) error {
			old, active := p.Name, h.store.Active
			p.Name = s
			if active == old {
				h.store.Active = s
			}
			if err := h.store.Save(); err != nil {
				p.Name, h.store.Active = old, active
				return err
			}
			h.refresh()
			h.profiles.Select(s)
			h.showDomains()
			h.a.Log(terminu.Success, "Renamed the profile "+old+" to "+s)
			return nil
		}))
}

func (h *hostu) copyProfile(p *profile.Profile) tea.Cmd {
	return h.a.Text(h.nameInput(terminu.TextInput{Title: "Copy", Subject: p.Name, Glyph: glyphCopy,
		Prompt: "Name of the copy", Suggest: h.unused(p.Name + " copy"), Verb: "copy"}, nil,
		func(s string) error {
			domains := make([]hosts.Domain, len(p.Domains))
			for i, d := range p.Domains {
				domains[i] = hosts.Domain{Name: d.Name, Addresses: slices.Clone(d.Addresses)}
			}
			if err := h.add(&profile.Profile{Name: s, Domains: domains}); err != nil {
				return err
			}
			h.a.Log(terminu.Success, "Copied the profile "+p.Name+" to "+s)
			return nil
		}))
}

// importProfile reads the hosts file again and keeps what is in it as a new
// profile; lines it could not read are told in a Warning toast.
func (h *hostu) importProfile() tea.Cmd {
	var domains []hosts.Domain
	var skipped int
	in := h.nameInput(terminu.TextInput{Title: "Import", Subject: h.env.Hosts, Glyph: glyphImport,
		Prompt: "Name of the profile made from it", Suggest: h.unused("imported"), Verb: "import"}, nil,
		func(s string) error {
			b, err := os.ReadFile(h.env.Hosts)
			if err != nil {
				return err
			}
			domains, skipped = hosts.Parse(string(b))
			return h.add(&profile.Profile{Name: s, Domains: domains})
		})
	in.Done = func(string) tea.Cmd {
		say := fmt.Sprintf("Imported %s from %s", count(len(domains), "domain"), h.env.Hosts)
		if skipped > 0 {
			return h.a.Toast(terminu.Warning, say+", skipped "+count(skipped, "line"))
		}
		return h.a.Toast(terminu.Success, say)
	}
	return h.a.Text(in)
}

func (h *hostu) deleteProfile(p *profile.Profile) tea.Cmd {
	i := slices.Index(h.store.Profiles, p)
	h.store.Remove(p)
	if err := h.store.Save(); err != nil {
		h.store.Profiles = slices.Insert(h.store.Profiles, i, p)
		return h.a.ErrorAbout("Can't save", h.store.Path(), sentence(err))
	}
	h.refresh()
	h.a.Log(terminu.Success, "Deleted the profile "+p.Name)
	return nil
}

// preview shows the hosts file p writes, as it writes it. Only looking is
// logged too (tdp note.7).
func (h *hostu) preview(p *profile.Profile) tea.Cmd {
	h.a.Log(terminu.Info, "Previewed the profile "+p.Name)
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
