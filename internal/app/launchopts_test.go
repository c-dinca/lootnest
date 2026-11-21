package app

import (
	"reflect"
	"testing"

	"lootnest/internal/core"
)

// A program started instead of the game may be the game itself under
// another name, so it's followed too.
func TestRunNames(t *testing.T) {
	o := core.LaunchOptions{StartWith: `D:\Games\RDR2\Game.EXE`}
	names := runNames(o, "rdr2")
	if !reflect.DeepEqual(names, []string{"rdr2", "game"}) {
		t.Fatalf("names = %v", names)
	}
	if got := runNames(core.LaunchOptions{StartWith: `D:\Games\RDR2\RDR2.exe`}, "rdr2"); len(got) != 1 {
		t.Fatalf("the game's own exe counted twice: %v", got)
	}
	if got := runNames(core.LaunchOptions{}, ""); len(got) != 0 {
		t.Fatalf("nothing to follow: %v", got)
	}
	if got := runNames(o, ""); !reflect.DeepEqual(got, []string{"game"}) {
		t.Fatalf("only the program: %v", got)
	}
	if liveName(names, map[string]bool{"game": true}) != "game" || liveName(names, map[string]bool{"rdr2": true, "game": true}) != "rdr2" || liveName(names, nil) != "" {
		t.Fatal("liveName")
	}
	if startedGrace(names) <= startedGrace([]string{"rdr2"}) {
		t.Fatal("a launcher gets longer to hand over")
	}
}
