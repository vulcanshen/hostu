// Package hosts reads and writes hosts files. hostu keeps a domain and the
// addresses it goes to; a hosts file has a line for an address and the names
// that go to it.
package hosts

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Domain is one name and the addresses it goes to, in the order written.
type Domain struct {
	Name      string   `yaml:"domain"`
	Addresses []string `yaml:"addresses,flow"`
}

// DefaultPath is the system's hosts file.
func DefaultPath() string {
	if runtime.GOOS == "windows" {
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = `C:\Windows`
		}
		return filepath.Join(root, "System32", "drivers", "etc", "hosts")
	}
	return "/etc/hosts"
}

// Parse reads a hosts file. What follows a # is a comment; a line is an
// address and the names that go to it, and a name on several lines has the
// addresses of all of them, in order. A line that is not an address and
// names is skipped, and counted.
func Parse(text string) (domains []Domain, skipped int) {
	at := map[string]int{} // a name in lower case → its place in domains
	for _, line := range strings.Split(text, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		if len(f) < 2 || CheckAddress(f[0]) != nil || slices.ContainsFunc(f[1:], func(n string) bool { return CheckDomain(n) != nil }) {
			skipped++
			continue
		}
		for _, name := range f[1:] {
			i, ok := at[strings.ToLower(name)]
			if !ok {
				i = len(domains)
				at[strings.ToLower(name)] = i
				domains = append(domains, Domain{Name: name})
			}
			if !slices.Contains(domains[i].Addresses, f[0]) {
				domains[i].Addresses = append(domains[i].Addresses, f[0])
			}
		}
	}
	return domains, skipped
}

// newline ends a hosts file's lines: Windows writes its own with CRLF.
var newline = map[bool]string{true: "\r\n", false: "\n"}[runtime.GOOS == "windows"]

// Format writes domains as the hosts file of the profile named profile: two
// lines of comment saying where it comes from, then a line for each address
// of each domain, the addresses padded to one width.
func Format(profile string, domains []Domain) string {
	w := 0
	for _, d := range domains {
		for _, a := range d.Addresses {
			w = max(w, len(a))
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Written by hostu from the profile %q.%s", profile, newline)
	b.WriteString("# hostu writes this whole file again when you activate a profile: change it in hostu." + newline)
	for _, d := range domains {
		for _, a := range d.Addresses {
			fmt.Fprintf(&b, "%-*s  %s%s", w, a, d.Name, newline)
		}
	}
	return b.String()
}

// Same reports whether two lists of domains make the same hosts file.
func Same(a, b []Domain) bool {
	return slices.EqualFunc(a, b, func(x, y Domain) bool {
		return x.Name == y.Name && slices.Equal(x.Addresses, y.Addresses)
	})
}

// Has reports whether domains has name, in any case.
func Has(domains []Domain, name string) bool {
	return slices.ContainsFunc(domains, func(d Domain) bool { return strings.EqualFold(d.Name, name) })
}

// CheckDomain says what is wrong with s as a name in a hosts file: letters,
// digits, -, _ and ., no part between dots empty or longer than 63, 253 at
// most in all. Empty is the caller's to say.
func CheckDomain(s string) error {
	if len(s) > 253 {
		return errors.New("A domain is at most 253 characters")
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			what := fmt.Sprintf("%q", r)
			if r == ' ' {
				what = "a space"
			}
			return fmt.Errorf("A domain has only letters, digits, -, _ and ., not %s", what)
		}
	}
	for _, part := range strings.Split(strings.TrimSuffix(s, "."), ".") {
		if part == "" {
			return errors.New("A domain has no empty part between its dots")
		}
		if len(part) > 63 {
			return fmt.Errorf("%s… is more than 63 characters between dots", part[:12])
		}
	}
	return nil
}

// CheckAddress says what is wrong with s as an IPv4 or IPv6 address (a zone
// allowed: fe80::1%lo0).
func CheckAddress(s string) error {
	if _, err := netip.ParseAddr(s); err != nil {
		return fmt.Errorf("%s is not an IPv4 or IPv6 address", s)
	}
	return nil
}

// Addresses reads the addresses typed in s, between spaces or commas, each
// once. Empty is the caller's to say.
func Addresses(s string) ([]string, error) {
	var out []string
	for _, a := range strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == ',' || r == '\t' }) {
		if err := CheckAddress(a); err != nil {
			return nil, err
		}
		if !slices.Contains(out, a) {
			out = append(out, a)
		}
	}
	return out, nil
}
