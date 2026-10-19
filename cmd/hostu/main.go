// hostu switches the hosts file between profiles, in the terminal.
package main

import (
	"fmt"
	"os"

	"github.com/vulcanshen/hostu/internal/place"
	"github.com/vulcanshen/hostu/internal/ui"
	"github.com/vulcanshen/hostu/internal/version"
)

const usage = `hostu: switch the hosts file between profiles

Usage:
  hostu            start
  hostu version    print the version (-v, --version)
  hostu help       print this help (-h, --help)

Profiles are kept in profiles.yaml in the settings directory
($XDG_CONFIG_HOME/hostu, else ~/.config/hostu; HOSTU__CONFIG moves it).
The hosts file hostu writes is the system's; HOSTU__HOSTS_FILE names
another.
`

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			fmt.Println(version.Display())
			return
		case "help", "-h", "--help":
			fmt.Print(usage)
			return
		default:
			fmt.Fprintf(os.Stderr, "hostu: unknown command %q\n\n%s", os.Args[1], usage)
			os.Exit(2)
		}
	}
	cfg, err := place.Config()
	if err == nil {
		var data string
		if data, err = place.Data(); err == nil {
			err = ui.New(ui.Env{Config: cfg, Data: data, Hosts: place.HostsFile()}).Run()
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "hostu:", err)
		os.Exit(1)
	}
}
