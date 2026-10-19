package ui

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/hostu/internal/hosts"
	"github.com/vulcanshen/hostu/internal/profile"
)

// domainActions are [2] DOMAINS': the domains of the profile under [1]'s
// cursor, a form each to add and to change one.
func (h *hostu) domainActions() []terminu.Action {
	none := func() bool { return h.current() == nil }
	return []terminu.Action{
		{Key: "Enter", Label: "Edit", Desc: "the domain and its addresses", Scope: terminu.ItemScope, Hint: "edit",
			Do: func() tea.Cmd { p, i := h.currentDomain(); return h.domainForm(p, i) }},
		{Key: "D", Label: "Delete", Desc: "take the domain out of the profile", Scope: terminu.ItemScope,
			AskFor: func() *terminu.Ask {
				p, i := h.currentDomain()
				return &terminu.Ask{Title: "Delete", Subject: p.Domains[i].Name, Verb: "delete",
					Question: "Delete " + p.Domains[i].Name + " from " + p.Name + "?"}
			},
			Do: func() tea.Cmd { p, i := h.currentDomain(); return h.deleteDomain(p, i) }},
		{Key: "n", Label: "new", Desc: "a domain and its addresses", Scope: terminu.PanelScope, Hint: "new", Hidden: none,
			Do: func() tea.Cmd { return h.domainForm(h.current(), -1) }},
	}
}

// currentDomain is the profile in [2] and the place of the domain under its
// cursor.
func (h *hostu) currentDomain() (*profile.Profile, int) {
	p := h.current()
	r, ok := h.domains.Current()
	if p == nil || !ok {
		return p, -1
	}
	return p, slices.IndexFunc(p.Domains, func(d hosts.Domain) bool { return d.Name == r.ID })
}

// domainForm is NEW DOMAIN, or EDIT for the domain at i: its name and its
// addresses, each typed in a text input.
func (h *hostu) domainForm(p *profile.Profile, i int) tea.Cmd {
	var name string
	var addrs []string
	f := &terminu.Form{Title: "New domain", Name: "New domain", Glyph: glyphDomain, Button: "Add"}
	if i >= 0 {
		name, addrs = p.Domains[i].Name, slices.Clone(p.Domains[i].Addresses)
		f.Title, f.Subject, f.Name, f.Button = "Edit", name, name, "Save"
	}
	typed := func() string { return strings.Join(addrs, " ") }
	f.Fields = []*terminu.FormField{
		{Label: "Domain", Value: func() string { return name }, Empty: "not set",
			Edit: func() terminu.Input {
				return terminu.TextInput{Title: "Domain", Glyph: glyphDomain, Prompt: "A name, as git.corp.local", Old: name,
					Check: func(s string) error {
						if s == "" {
							return terminu.ErrEmpty
						}
						return hosts.CheckDomain(s)
					},
					Done: func(s string) tea.Cmd { name = s; return nil }}
			},
			Check: func() string {
				switch {
				case name == "":
					return "Domain is required"
				case slices.ContainsFunc(p.Domains, func(d hosts.Domain) bool {
					return strings.EqualFold(d.Name, name) && (i < 0 || d.Name != p.Domains[i].Name)
				}):
					return name + " is in " + p.Name + " already"
				}
				return ""
			}},
		{Label: "Addresses", Value: typed, Empty: "not set",
			Edit: func() terminu.Input {
				return terminu.TextInput{Title: "Addresses", Glyph: glyphDomain, Prompt: "IPv4 or IPv6 addresses, between spaces",
					Old: typed(),
					Check: func(s string) error {
						if strings.TrimSpace(s) == "" {
							return terminu.ErrEmpty
						}
						_, err := hosts.Addresses(s)
						return err
					},
					Done: func(s string) tea.Cmd { addrs, _ = hosts.Addresses(s); return nil }}
			},
			Check: func() string {
				if len(addrs) == 0 {
					return "Addresses are required"
				}
				return ""
			}},
	}
	f.Submit = func() (tea.Cmd, error) {
		d := hosts.Domain{Name: name, Addresses: addrs}
		old := slices.Clone(p.Domains)
		if i < 0 {
			p.Domains = append(p.Domains, d)
		} else {
			p.Domains[i] = d
		}
		if err := h.save(); err != nil {
			p.Domains = old
			return nil, err
		}
		h.refresh()
		h.domains.Select(name)
		if i < 0 {
			// Added: the menu it may have come from was about the domain
			// under the cursor before (tdp T1).
			return h.a.CloseAll(), nil
		}
		return nil, nil
	}
	return h.a.Form(f)
}

func (h *hostu) deleteDomain(p *profile.Profile, i int) tea.Cmd {
	old := slices.Clone(p.Domains)
	p.Domains = slices.Delete(p.Domains, i, i+1)
	if err := h.save(); err != nil {
		p.Domains = old
		return h.a.ErrorAbout("Can't save", h.store.Path(), sentence(err))
	}
	h.refresh()
	return nil
}
