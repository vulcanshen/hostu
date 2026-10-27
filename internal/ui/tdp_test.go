package ui

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/tdptest"

	"github.com/vulcanshen/hostu/internal/hosts"
	"github.com/vulcanshen/hostu/internal/place"
	"github.com/vulcanshen/hostu/internal/profile"
)

// newApp is hostu as main makes it, all but Run: its places from the
// environment.
func newApp() *terminu.App {
	cfg, err := place.Config()
	if err != nil {
		panic(err)
	}
	data, err := place.Data()
	if err != nil {
		panic(err)
	}
	return New(Env{Config: cfg, Data: data, Hosts: place.HostsFile()})
}

// The family's walk (terminu-go's tdptest): hostu launched as for the first
// time on many terminal sizes and icon widths, on each panel the Space menu,
// the global operation popup, the App Log and the key reference opened and
// closed, then a terminal under the least size; every frame each row exactly
// the terminal's width, no control character sent to it, no colour of
// hostu's own drawn as text (tdp L4, L1, E5). The hosts file is one in the
// sandbox, never the system's.
func TestTDP(t *testing.T) {
	tdptest.Check(t, tdptest.App{Name: "hostu", New: newApp,
		Env: func(home string) map[string]string {
			return map[string]string{"HOSTU__HOSTS_FILE": filepath.Join(home, "hosts")}
		},
		Seed: func(home string) {
			if err := os.WriteFile(filepath.Join(home, "hosts"), []byte(systemHosts), 0o644); err != nil {
				panic(err)
			}
		}})
}

// The same walk with values that have broken screens: every tricky one a
// profile can be named is a profile's name, the active one the first, and
// the hosts file's name has a Tab and a colour code (components, control
// characters), but on Windows, where a file name can't have them. A colour
// code in a name there is the data's, drawn by its name as it must be
// (Tricky).
func TestTDPWithTrickyValues(t *testing.T) {
	name := "ho\x1b[31mst\ts"
	if runtime.GOOS == "windows" {
		name = "hosts"
	}
	hostsFile := func(home string) string { return filepath.Join(home, name) }
	tdptest.Check(t, tdptest.App{Name: "hostu", New: newApp, Tricky: true,
		Env: func(home string) map[string]string { return map[string]string{"HOSTU__HOSTS_FILE": hostsFile(home)} },
		Seed: func(home string) {
			if err := os.WriteFile(hostsFile(home), []byte(systemHosts), 0o644); err != nil {
				panic(err)
			}
			s, _, err := profile.Load(filepath.Join(home, ".hostu"))
			if err != nil {
				panic(err)
			}
			for _, v := range tdptest.Tricky {
				if v == "" || profile.CheckName(v) != nil || s.Taken(v, nil) {
					continue
				}
				s.Profiles = append(s.Profiles, &profile.Profile{Name: v,
					Domains: []hosts.Domain{{Name: "nas.lan", Addresses: []string{"10.0.0.9"}}}})
			}
			s.Active = s.Profiles[0].Name
			if err := s.Save(); err != nil {
				panic(err)
			}
		}})
}
