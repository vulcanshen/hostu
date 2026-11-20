// hostu switches the hosts file between profiles, in the terminal.
package main

import (
	"fmt"
	"io"
	"os"

	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/hostu/internal/place"
	"github.com/vulcanshen/hostu/internal/ui"
	"github.com/vulcanshen/hostu/internal/version"
)

const usage = `hostu: switch the hosts file between profiles

Usage:
  hostu            start
  hostu update     update to the latest release, the way it was installed
  hostu version    print the version (-v, --version)
  hostu help       print this help (-h, --help)

Profiles are kept in profiles.yaml in the data directory (~/.hostu;
HOSTU__DATA moves it). The hosts file hostu writes is the system's;
HOSTU__HOSTS_FILE names another.
`

func main() {
	if len(os.Args) > 1 {
		os.Exit(command(os.Args[1:], os.Stdout, os.Stderr))
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

// command runs hostu's command args and gives its exit status: results on
// stdout; errors on stderr after "hostu: ", a wrong use with the usage and
// status 2 (tdp E1).
func command(args []string, stdout, stderr io.Writer) int {
	switch args[0] {
	case "version", "-v", "--version":
		fmt.Fprintln(stdout, version.Display())
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
	case "update":
		return terminu.SelfUpdate{App: "hostu", Version: version.Version, Owner: "vulcanshen", Tap: "vulcanshen/tap",
			Source: "git pull && make install"}.Run(stdout, stderr)
	default:
		fmt.Fprintf(stderr, "hostu: Unknown command %q\n\n%s", args[0], usage)
		return 2
	}
	return 0
}
