package core

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf16"
)

// ---------------------------------------------------------------------------
// Keep a game offline: Windows Firewall rules that stop a game's programs
// from connecting out, made (and taken away) through Windows' own
// NetSecurity commands, as administrator. Lootnest asks Windows; Windows asks
// the user. The rules are in the "Lootnest" group, so they're easy to find
// in Windows' own firewall settings too.
// ---------------------------------------------------------------------------

// MaxOfflineExes is how many of a game's programs get a rule.
const MaxOfflineExes = 30

var skipExeDirs = map[string]bool{"_commonredist": true, "redist": true, "redistributables": true, "directx": true, "vcredist": true, "dotnet": true, "__installer": true, "installers": true, "support": true}

// GameExes lists the programs in a game's folder (a few levels down, as
// Unreal games keep theirs in <Game>\Binaries\Win64), main first, without
// redistributable installers and uninstallers.
func GameExes(dir, main string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		k := strings.ToLower(filepath.Clean(p))
		if !seen[k] && len(out) < MaxOfflineExes {
			seen[k] = true
			out = append(out, filepath.Clean(p))
		}
	}
	if main != "" && strings.EqualFold(filepath.Ext(main), ".exe") {
		add(main)
	}
	if dir == "" {
		return out
	}
	root := filepath.Clean(dir)
	var found []string
	n := 0
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		n++
		if n > 50000 {
			return filepath.SkipAll
		}
		if err != nil {
			return filepath.SkipDir
		}
		name := strings.ToLower(d.Name())
		if d.IsDir() {
			if p != root && (skipExeDirs[name] || strings.HasPrefix(name, ".lootnest")) {
				return filepath.SkipDir
			}
			if rel, _ := filepath.Rel(root, p); strings.Count(rel, string(filepath.Separator)) >= 4 {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(name) != ".exe" || strings.HasPrefix(name, "unins") || strings.HasPrefix(name, "vc_redist") || strings.HasPrefix(name, "vcredist") || name == "dxsetup.exe" || strings.HasPrefix(name, "dotnet") {
			return nil
		}
		found = append(found, p)
		return nil
	})
	sort.Slice(found, func(i, j int) bool {
		// Shallower first: the game's own exe before its helpers.
		di, dj := strings.Count(found[i], string(filepath.Separator)), strings.Count(found[j], string(filepath.Separator))
		if di != dj {
			return di < dj
		}
		return strings.ToLower(found[i]) < strings.ToLower(found[j])
	})
	for _, p := range found {
		add(p)
	}
	return out
}

// OfflineTag names one game's rules: "lootnest:" and a short hash of its
// Library key, in each rule's description.
func OfflineTag(key string) string {
	h := sha1.Sum([]byte(strings.ToLower(key)))
	return "lootnest:" + hex.EncodeToString(h[:6])
}

func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// FirewallScript is the PowerShell that blocks (block true) or lets go
// online again a game's programs. It first takes away the game's rules, so
// running it twice leaves one set.
func FirewallScript(tag, name string, exes []string, block bool) string {
	var b strings.Builder
	b.WriteString("$ErrorActionPreference = 'Stop'\n")
	b.WriteString("Get-NetFirewallRule -Group 'Lootnest' -ErrorAction SilentlyContinue | Where-Object { $_.Description -eq " + psQuote(tag) + " } | Remove-NetFirewallRule\n")
	if block {
		for _, e := range exes {
			base := e[strings.LastIndexAny(e, `\/`)+1:]
			label := "Lootnest keeps " + name + " offline (" + base + ")"
			b.WriteString("New-NetFirewallRule -DisplayName " + psQuote(label) + " -Group 'Lootnest' -Description " + psQuote(tag) +
				" -Direction Outbound -Action Block -Profile Any -Program " + psQuote(e) + " | Out-Null\n")
		}
	}
	return b.String()
}

// EncodePowerShell gives a script as powershell -EncodedCommand takes it
// (base64 of UTF-16LE), so no quoting reaches the command line.
func EncodePowerShell(script string) string {
	u := utf16.Encode([]rune(script))
	raw := make([]byte, len(u)*2)
	for i, c := range u {
		binary.LittleEndian.PutUint16(raw[i*2:], c)
	}
	return base64.StdEncoding.EncodeToString(raw)
}
