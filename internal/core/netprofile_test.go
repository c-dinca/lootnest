package core

import (
	"strings"
	"testing"
)

func TestNetworkCategoryScripts(t *testing.T) {
	s, err := NetworkCategoryScript("192.168.1.20")
	if err != nil || !strings.Contains(s, "-IPAddress '192.168.1.20'") || !strings.Contains(s, "NetworkCategory") {
		t.Fatalf("%q %v", s, err)
	}
	m, err := MakePrivateScript("192.168.1.20")
	if err != nil || !strings.Contains(m, "-NetworkCategory Private") || !strings.Contains(m, "exit 10") {
		t.Fatalf("%q %v", m, err)
	}
	for _, bad := range []string{"", "1.2.3", "1.2.3.4'; Remove-Item C:\\", "lootnest"} {
		if _, err := NetworkCategoryScript(bad); err == nil {
			t.Fatalf("took %q", bad)
		}
		if _, err := MakePrivateScript(bad); err == nil {
			t.Fatalf("took %q", bad)
		}
	}
}

func TestNetworkCategory(t *testing.T) {
	for out, want := range map[string]string{
		"Public\r\n":                     "Public",
		"Private":                        "Private",
		"warning\nDomainAuthenticated\n": "DomainAuthenticated",
		"":                               "",
		"Get-NetIPAddress : No matching MSFT_NetIPAddress objects": "",
	} {
		if got := NetworkCategory(out); got != want {
			t.Fatalf("%q: %q, want %q", out, got, want)
		}
	}
}
