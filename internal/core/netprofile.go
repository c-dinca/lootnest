package core

import (
	"errors"
	"net"
	"strings"
)

// Windows sorts each network into Public, Private or Domain. The streaming
// engine's firewall rules allow Private networks only (ApolloInstallScript),
// so on a network Windows calls Public a TV can't reach the PC. Lootnest
// reads the category of the network a TV is on, and makes it Private only
// when the user asks.

// NetworkCategoryScript prints the category (Public, Private or
// DomainAuthenticated) of the network that has the PC address ip.
func NetworkCategoryScript(ip string) (string, error) {
	ip, err := plainIP(ip)
	if err != nil {
		return "", err
	}
	return `$ErrorActionPreference = 'Stop'
$i = (Get-NetIPAddress -IPAddress ` + psQuote(ip) + `).InterfaceIndex
[string](Get-NetConnectionProfile -InterfaceIndex $i).NetworkCategory
`, nil
}

// MakePrivateScript makes the network with the PC address ip Private. It
// runs as administrator; exit 10 means Windows didn't change it.
func MakePrivateScript(ip string) (string, error) {
	ip, err := plainIP(ip)
	if err != nil {
		return "", err
	}
	return `$ErrorActionPreference = 'Stop'
try {
  $i = (Get-NetIPAddress -IPAddress ` + psQuote(ip) + `).InterfaceIndex
  Set-NetConnectionProfile -InterfaceIndex $i -NetworkCategory Private
} catch { exit 10 }
exit 0
`, nil
}

// NetworkCategory reads NetworkCategoryScript's output: "Public",
// "Private", "DomainAuthenticated", or "" when it can't tell.
func NetworkCategory(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	switch last {
	case "Public", "Private", "DomainAuthenticated":
		return last
	}
	return ""
}

// PublicNetworkError is a TV's network that Windows calls Public.
type PublicNetworkError struct{ PC string }

func (e *PublicNetworkError) Error() string {
	return "Windows treats this network as public, so the TV can't reach this PC. Make it private, then install again"
}

func plainIP(ip string) (string, error) {
	p := net.ParseIP(strings.TrimSpace(ip))
	if p == nil {
		return "", errors.New("not an address: " + ip)
	}
	return p.String(), nil
}
