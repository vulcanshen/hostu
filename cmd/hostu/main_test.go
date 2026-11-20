package main

import (
	"bytes"
	"strings"
	"testing"
)

// run is command with what it printed.
func run(args ...string) (status int, stdout, stderr string) {
	var out, errs bytes.Buffer
	status = command(args, &out, &errs)
	return status, out.String(), errs.String()
}

// version and help print on stdout and are done (tdp E1).
func TestVersionAndHelp(t *testing.T) {
	for _, arg := range []string{"version", "-v", "--version"} {
		if status, out, errs := run(arg); status != 0 || out != "dev\n" || errs != "" {
			t.Fatalf("%s: %d %q %q", arg, status, out, errs)
		}
	}
	for _, arg := range []string{"help", "-h", "--help"} {
		if status, out, errs := run(arg); status != 0 || out != usage || errs != "" {
			t.Fatalf("%s: %d %q %q", arg, status, out, errs)
		}
	}
	if !strings.Contains(usage, "hostu update") {
		t.Fatal("the usage leaves out update")
	}
}

// A command hostu doesn't have is a wrong use: what is wrong after "hostu: ",
// capitalised, a blank line and the usage on stderr, status 2 (tdp E1).
func TestUnknownCommand(t *testing.T) {
	status, out, errs := run("upgrade")
	if status != 2 || out != "" || errs != "hostu: Unknown command \"upgrade\"\n\n"+usage {
		t.Fatalf("%d %q %q", status, out, errs)
	}
}

// update in a build from source says to update it there, after "hostu: ",
// and is not done, status 1; it asks the network nothing (tdp E1).
func TestUpdateFromSource(t *testing.T) {
	status, out, errs := run("update")
	if status != 1 || out != "" || !strings.HasPrefix(errs, "hostu: ") || !strings.Contains(errs, "git pull && make install") {
		t.Fatalf("%d %q %q", status, out, errs)
	}
}
