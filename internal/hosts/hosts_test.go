package hosts

import (
	"reflect"
	"strings"
	"testing"
)

const macHosts = `##
# Host Database
#
# localhost is used to configure the loopback interface
##
127.0.0.1	localhost
255.255.255.255	broadcasthost
::1             localhost
10.0.0.5 git.corp.local  ci.corp.local # the build machines
not-an-address  web.local
10.0.0.6
10.0.0.7 bad~name
fe80::1%lo0 localhost
`

func TestParse(t *testing.T) {
	got, skipped := Parse(macHosts)
	want := []Domain{
		{Name: "localhost", Addresses: []string{"127.0.0.1", "::1", "fe80::1%lo0"}},
		{Name: "broadcasthost", Addresses: []string{"255.255.255.255"}},
		{Name: "git.corp.local", Addresses: []string{"10.0.0.5"}},
		{Name: "ci.corp.local", Addresses: []string{"10.0.0.5"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsed\n%+v\nwant\n%+v", got, want)
	}
	if skipped != 3 {
		t.Fatalf("skipped %d lines, want 3", skipped)
	}
}

// What Format writes, Parse reads back the same: the active profile and the
// hosts file it wrote agree.
func TestFormatReadsBack(t *testing.T) {
	domains, _ := Parse(macHosts)
	text := Format("office", domains)
	if !strings.HasPrefix(text, "# Written by hostu from the profile \"office\".") {
		t.Fatalf("no header:\n%s", text)
	}
	if !strings.Contains(text, "255.255.255.255  broadcasthost") || !strings.Contains(text, "::1              localhost") {
		t.Fatalf("the addresses are not padded to one width:\n%s", text)
	}
	back, skipped := Parse(text)
	if !Same(back, domains) || skipped != 0 {
		t.Fatalf("read back %+v (%d skipped), wrote %+v", back, skipped, domains)
	}
}

func TestCheckDomain(t *testing.T) {
	for _, ok := range []string{"localhost", "git.corp.local", "my_host", "a-b.c", "fqdn.example.com."} {
		if err := CheckDomain(ok); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	for s, want := range map[string]string{
		"git corp":                      "not a space",
		"bad~name":                      `not '~'`,
		"a..b":                          "no empty part",
		".a":                            "no empty part",
		strings.Repeat("x", 64) + ".a":  "more than 63",
		strings.Repeat("a.", 127) + "a": "at most 253",
	} {
		if err := CheckDomain(s); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%.20s…: %v, want %q", s, err, want)
		}
	}
}

func TestAddresses(t *testing.T) {
	got, err := Addresses(" 10.1.2.3,fd00::3  10.1.2.3 fe80::1%en0 ")
	if err != nil || !reflect.DeepEqual(got, []string{"10.1.2.3", "fd00::3", "fe80::1%en0"}) {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := Addresses("10.1.2.3 10.1.2"); err == nil || err.Error() != "10.1.2 is not an IPv4 or IPv6 address" {
		t.Fatalf("a wrong address: %v", err)
	}
}
