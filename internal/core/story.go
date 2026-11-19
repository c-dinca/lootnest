package core

import (
	"encoding/json"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Story is how far a game's story has come, read from the achievements a
// game gives for finishing its chapters ("Complete Chapter 3.", "Leave Act
// II") and for finishing it ("Finish the game", "Reach the ... ending").
// Nothing is read from the game itself: a game whose achievements don't
// mark its chapters has no story here.
type Story struct {
	Percent  int         `json:"percent"`            // 0 to 100
	Finished bool        `json:"finished,omitempty"` // the last step is done
	Steps    []StoryStep `json:"steps"`              // in the story's order
}

// StoryStep is one marked point of a story.
type StoryStep struct {
	Label string `json:"label"` // "Prologue", "Chapter 3", "Epilogue", "The end"
	Done  bool   `json:"done"`
	unit  float64
}

var (
	// "Complete Chapter 1.", "Finished Chapter 12 of The Dark Awakens.",
	// "Leave Act II", "Clear Mission 03.", "Complete the first chapter".
	storyStepRe = regexp.MustCompile(`^(?:complete|completed|finish|finished|clear|cleared|beat|beaten|leave|left|survive|survived)\s+(?:the\s+)?(?:(first|second|third|fourth|fifth|sixth|seventh|eighth|ninth|tenth)\s+)?(chapter|act|part|episode|mission|stage)\b\s*(\S*)\s*(.*)$`)
	// "Clear the Prologue Mission.", "Complete the Epilogue."
	storyEdgeRe = regexp.MustCompile(`^(?:complete|completed|finish|finished|clear|cleared|beat|beaten|reach|reached)\s+(?:the\s+)?(prologue|epilogue)\b\s*(.*)$`)
	// "Finish the game on any difficulty.", "Complete the main story on
	// Standard mode or higher.", "Completed the final mission."
	storyEndRe = regexp.MustCompile(`^(?:complete|completed|finish|finished|beat|beaten|clear|cleared)\s+(?:the\s+)?(?:game|story|main story|main quest|campaign|main campaign|final mission|final chapter|last chapter)\b`)
	// "Achieved the "Elden Lord" ending", "Reach "To Link the First Flame"
	// ending.", "Attained the "Return" ending".
	storyEndingRe = regexp.MustCompile(`^(?:achieve|achieved|attain|attained|reach|reached|see|saw|unlock|unlocked|get|got)\s+(?:the\s+|an?\s+)?(?:"[^"]*"\s+|\S+\s+)?ending$`)
	// What makes a chapter's achievement a challenge rather than the
	// chapter itself: "Complete Chapter 1 B-Side", "... without dying",
	// "... on Hard", "Collect all ... in Chapter 2".
	storyNotStep = regexp.MustCompile(`\b(?:without|with|under|within|less|side|mode|difficulty|rank|all|every|only|hard|speed|time|damage|minutes?|hours?|seconds?|co-?op|online)\b|%`)
	// Rests that only say whose chapter it is ("of The Dark Awakens") or
	// add a line of flavour ("- for somewhere altogether darker").
	storyOfRe = regexp.MustCompile(`^(?:(?:of|in)\s|[-\x{2013}\x{2014}:(])`)
)

var storyOrdinals = map[string]int{"first": 1, "second": 2, "third": 3, "fourth": 4, "fifth": 5, "sixth": 6, "seventh": 7, "eighth": 8, "ninth": 9, "tenth": 10}
var storyWords = map[string]int{"one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10, "eleven": 11, "twelve": 12}

// storyNumber reads "3", "03", "II" or "three"; 0 when it isn't one.
func storyNumber(s string) int {
	s = strings.Trim(s, ".,:;!'\"")
	if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 99 {
		return n
	}
	if n := storyWords[s]; n > 0 {
		return n
	}
	return roman(s)
}

func roman(s string) int {
	vals := map[rune]int{'i': 1, 'v': 5, 'x': 10, 'l': 50}
	n, prev := 0, 0
	if s == "" || len(s) > 7 {
		return 0
	}
	for i := len(s) - 1; i >= 0; i-- {
		v := vals[rune(s[i])]
		if v == 0 {
			return 0
		}
		if v < prev {
			n -= v
		} else {
			n += v
			prev = v
		}
	}
	if n > 60 {
		return 0
	}
	return n
}

// storyText is an achievement's words as the patterns read them.
func storyText(a Achievement) string {
	s := strings.TrimSpace(a.Description)
	if s == "" {
		s = a.Name
	}
	s = strings.ToLower(s)
	s = strings.NewReplacer("“", `"`, "”", `"`, "‘", "'", "’", "'", " ", " ").Replace(s)
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimRight(s, ".! ")
}

type storyMark struct {
	kind string // chapter, act...; "prologue", "epilogue", "end"
	n    int
	done bool
}

func storyMarkOf(a Achievement) (storyMark, bool) {
	t := storyText(a)
	if t == "" {
		return storyMark{}, false
	}
	if storyEndRe.MatchString(t) || storyEndingRe.MatchString(t) {
		return storyMark{kind: "end", done: a.Unlocked}, true
	}
	if m := storyEdgeRe.FindStringSubmatch(t); m != nil {
		rest := strings.TrimSpace(m[2])
		if rest == "" || rest == "mission" || rest == "chapter" || rest == "of the story" || rest == "of the game" {
			return storyMark{kind: m[1], done: a.Unlocked}, true
		}
		return storyMark{}, false
	}
	m := storyStepRe.FindStringSubmatch(t)
	if m == nil {
		return storyMark{}, false
	}
	n := storyOrdinals[m[1]]
	rest := strings.TrimSpace(m[3] + " " + m[4])
	if n == 0 {
		n = storyNumber(m[3])
		rest = strings.TrimSpace(m[4])
	}
	if n == 0 {
		return storyMark{}, false
	}
	// Whatever follows may only name the story ("of The Dark Awakens").
	if rest != "" && (!storyOfRe.MatchString(rest) || storyNotStep.MatchString(rest)) {
		return storyMark{}, false
	}
	return storyMark{kind: m[2], n: n, done: a.Unlocked}, true
}

// StoryOf reads a game's story from its achievements, or nil when they
// don't mark it: at least two numbered chapters (acts, parts...) of one
// kind, or an ending that's been reached.
func StoryOf(list []Achievement) *Story {
	byKind := map[string]map[int]bool{} // kind -> number -> done
	var prologue, epilogue, end *bool
	set := func(p **bool, done bool) {
		if *p == nil {
			v := done
			*p = &v
		} else if done {
			**p = true
		}
	}
	for _, a := range list {
		m, ok := storyMarkOf(a)
		if !ok {
			continue
		}
		switch m.kind {
		case "end":
			set(&end, m.done)
		case "prologue":
			set(&prologue, m.done)
		case "epilogue":
			set(&epilogue, m.done)
		default:
			if byKind[m.kind] == nil {
				byKind[m.kind] = map[int]bool{}
			}
			byKind[m.kind][m.n] = byKind[m.kind][m.n] || m.done
		}
	}
	// The kind with the most numbered steps is the story's; the others are
	// side stories ("Act 1" of an online heist beside "Chapter 3").
	kind := ""
	for k, ns := range byKind {
		if kind == "" || len(ns) > len(byKind[kind]) || (len(ns) == len(byKind[kind]) && k < kind) {
			kind = k
		}
	}
	nums := byKind[kind]
	if len(nums) < 2 {
		// No chapters to count: only whether it's been finished.
		if end != nil && *end {
			return &Story{Percent: 100, Finished: true, Steps: []StoryStep{{Label: "The end", Done: true}}}
		}
		return nil
	}
	var steps []StoryStep
	order := make([]int, 0, len(nums))
	for n := range nums {
		order = append(order, n)
	}
	sort.Ints(order)
	if prologue != nil && order[0] >= 1 {
		steps = append(steps, StoryStep{Label: "Prologue", Done: *prologue, unit: 0.5})
	}
	name := strings.ToUpper(kind[:1]) + kind[1:]
	for _, n := range order {
		steps = append(steps, StoryStep{Label: name + " " + strconv.Itoa(n), Done: nums[n], unit: float64(n)})
	}
	last := float64(order[len(order)-1])
	switch {
	case end != nil: // the game's own end; an epilogue after it is extra
		steps = append(steps, StoryStep{Label: "The end", Done: *end, unit: last + 1})
	case epilogue != nil:
		steps = append(steps, StoryStep{Label: "Epilogue", Done: *epilogue, unit: last + 1})
	}
	// A later step done means the ones before it are too.
	reached := -1
	for i, s := range steps {
		if s.Done {
			reached = i
		}
	}
	for i := 0; i <= reached; i++ {
		steps[i].Done = true
	}
	st := &Story{Steps: steps}
	final := steps[len(steps)-1]
	switch {
	case final.Done:
		st.Percent, st.Finished = 100, true
	case reached >= 0:
		st.Percent = int(math.Round(100 * steps[reached].unit / final.unit))
		if st.Percent > 99 {
			st.Percent = 99
		}
	}
	return st
}

// StoryWords fills in the words of achievements a list has without them
// (Steam keeps hidden ones' descriptions back), from Steam's public list
// of the game's achievements, so StoryOf can read them. The list given
// isn't changed.
func StoryWords(list []Achievement, words map[string]Achievement) []Achievement {
	if len(words) == 0 {
		return list
	}
	out := make([]Achievement, len(list))
	copy(out, list)
	for i := range out {
		w, ok := words[out[i].ID]
		if !ok {
			continue
		}
		if strings.TrimSpace(out[i].Description) == "" {
			out[i].Description = w.Description
		}
		if strings.TrimSpace(out[i].Name) == "" || out[i].Name == out[i].ID {
			out[i].Name = w.Name
		}
	}
	return out
}

// ParsePublicAchievements reads Steam's public list of a game's
// achievements (IPlayerService/GetGameAchievements: no key, hidden ones'
// descriptions included), by id.
func ParsePublicAchievements(b []byte) map[string]Achievement {
	var r struct {
		Response struct {
			Achievements []struct {
				ID   string `json:"internal_name"`
				Name string `json:"localized_name"`
				Desc string `json:"localized_desc"`
			} `json:"achievements"`
		} `json:"response"`
	}
	if json.Unmarshal(b, &r) != nil || len(r.Response.Achievements) == 0 {
		return nil
	}
	out := make(map[string]Achievement, len(r.Response.Achievements))
	for _, a := range r.Response.Achievements {
		if a.ID != "" {
			out[a.ID] = Achievement{ID: a.ID, Name: a.Name, Description: a.Desc}
		}
	}
	return out
}
