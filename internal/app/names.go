package app

import (
	"strconv"
	"strings"
	"unicode"

	"lootnest/internal/core"
)

// trimRunes is s cut to at most n runes, without spaces at either end.
func trimRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		s = string(r[:n])
	}
	return strings.TrimSpace(s)
}

// trimWords is trimRunes that cuts before a word it would split, when the
// cut leaves at least one word.
func trimWords(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n || n == 0 || !unicode.IsLetter(r[n]) && !unicode.IsDigit(r[n]) {
		return strings.TrimRight(trimRunes(s, n), " ,")
	}
	cut := string(r[:n])
	if i := strings.LastIndex(cut, " "); i > 0 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,")
}

// freeName is name, or name with " 2", " 3" and so on when another profile
// has it, kept within 24 runes.
func freeName(name string, players []core.Player) string {
	taken := func(s string) bool {
		for _, p := range players {
			if strings.EqualFold(p.Name, s) {
				return true
			}
		}
		return false
	}
	out := name
	for i := 2; taken(out); i++ {
		n := " " + strconv.Itoa(i)
		out = trimWords(name, 24-len(n)) + n
	}
	return out
}
