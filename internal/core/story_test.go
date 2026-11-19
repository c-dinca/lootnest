package core

import "testing"

// achs makes a list from descriptions; the first `done` are unlocked.
func achs(done int, descs ...string) []Achievement {
	out := make([]Achievement, len(descs))
	for i, d := range descs {
		out[i] = Achievement{ID: "a" + string(rune('A'+i)), Name: "Name", Description: d, Unlocked: i < done}
	}
	return out
}

func labels(s *Story) (all string) {
	for _, st := range s.Steps {
		mark := " "
		if st.Done {
			mark = "+"
		}
		all += mark + st.Label + ";"
	}
	return all
}

func TestStoryChapters(t *testing.T) {
	// Red Dead Redemption 2's words: chapters, then the epilogue.
	list := achs(3, "Complete Chapter 1.", "Complete Chapter 2.", "Complete Chapter 3.", "Complete Chapter 4.", "Complete Chapter 5.", "Complete the Epilogue.",
		"Complete all optional Honor story missions.", "Earn 70 Gold Medals in Story missions.", "Red Dead Online: Take part in a Series.")
	s := StoryOf(list)
	if s == nil || s.Percent != 50 || s.Finished {
		t.Fatalf("story = %+v", s)
	}
	if got, want := labels(s), "+Chapter 1;+Chapter 2;+Chapter 3; Chapter 4; Chapter 5; Epilogue;"; got != want {
		t.Errorf("steps = %q, want %q", got, want)
	}
	for i := range list {
		list[i].Unlocked = true
	}
	if s := StoryOf(list); !s.Finished || s.Percent != 100 {
		t.Errorf("all done = %+v", s)
	}
}

func TestStoryActsAndEnd(t *testing.T) {
	// Baldur's Gate 3: roman acts with a line of flavour, then the end.
	list := achs(2, "Leave Act I - for somewhere altogether darker.", "Leave Act II - Baldur's Gate is just over the horizon.",
		"Finish the game (with a heartfelt 'thank you!' from Larian Studios).", "Complete the game in Honour mode.")
	s := StoryOf(list)
	if s == nil || s.Percent != 67 || labels(s) != "+Act 1;+Act 2; The end;" {
		t.Fatalf("story = %+v %s", s, labels(s))
	}
	// Finishing on any difficulty finishes the story.
	list[3].Unlocked = true
	if s := StoryOf(list); !s.Finished || labels(s) != "+Act 1;+Act 2;+The end;" {
		t.Errorf("finished on Honour = %+v", s)
	}
}

func TestStoryChallengesAreNotChapters(t *testing.T) {
	s := StoryOf(achs(6, "Complete Chapter 1", "Complete Chapter 1 B-Side", "Collect the Crystal Heart in Chapter 2", "Complete Chapter 2",
		"Clear Mission 07 with both Nero and V.", "Finished Chapter 15 of The Dark Awakens with Jin Kazama victorious.", "Complete Chapter 3", "Complete Chapter 4"))
	if s == nil || labels(s) != "+Chapter 1;+Chapter 2; Chapter 3; Chapter 4;" || s.Percent != 50 {
		t.Fatalf("story = %+v %s", s, labels(s))
	}
}

func TestStoryPrologueAndMissions(t *testing.T) {
	// Devil May Cry 5: zero-padded missions after a prologue.
	s := StoryOf(achs(2, "Clear the Prologue Mission.", "Clear Mission 03.", "Clear Mission 06.", "Complete the game on Human mode."))
	if s == nil || labels(s) != "+Prologue;+Mission 3; Mission 6; The end;" || s.Percent != 43 {
		t.Fatalf("story = %+v %s", s, labels(s))
	}
}

func TestStoryOnlyAnEnding(t *testing.T) {
	// Elden Ring marks only its endings: finished or nothing.
	if s := StoryOf(achs(0, `Achieved the "Elden Lord" ending`, "Defeated Margit, the Fell Omen")); s != nil {
		t.Errorf("not finished = %+v", s)
	}
	s := StoryOf(achs(1, `Achieved the "Elden Lord" ending`, "Defeated Margit, the Fell Omen"))
	if s == nil || !s.Finished || s.Percent != 100 {
		t.Errorf("finished = %+v", s)
	}
	for _, d := range []string{`Reach "To Link the First Flame" ending.`, "Completed the final mission.", "Finish the story on at least Casual difficulty.", "Complete the main quest in the story"} {
		if s := StoryOf(achs(1, d)); s == nil || !s.Finished {
			t.Errorf("%q isn't an ending", d)
		}
	}
	for _, d := range []string{"Complete all chapters of the 'Drift Club' Horizon Story", "Complete the first chapter in the Icons of Speed Story", "Defeat Baldur", "Complete Part 1"} {
		if s := StoryOf(achs(1, d)); s != nil {
			t.Errorf("%q read as %+v", d, s)
		}
	}
}

func TestStoryNumbers(t *testing.T) {
	for in, want := range map[string]int{"1": 1, "03": 3, "ii": 2, "iv": 4, "ix": 9, "xii": 12, "three": 3, "1.": 1, "": 0, "boss": 0, "100": 0} {
		if got := storyNumber(in); got != want {
			t.Errorf("storyNumber(%q) = %d, want %d", in, got, want)
		}
	}
	// "the first chapter" counts as Chapter 1.
	s := StoryOf(achs(1, "Complete the first chapter", "Complete the second chapter"))
	if s == nil || labels(s) != "+Chapter 1; Chapter 2;" {
		t.Errorf("ordinals = %+v", s)
	}
}

func TestStoryWordsFillHidden(t *testing.T) {
	list := []Achievement{{ID: "Achievement1", Name: "Back in the Mud", Hidden: true, Unlocked: true}, {ID: "Achievement2", Name: "Achievement2", Hidden: true}}
	words := ParsePublicAchievements([]byte(`{"response":{"achievements":[
		{"internal_name":"Achievement1","localized_name":"Back in the Mud","localized_desc":"Complete Chapter 1.","hidden":true},
		{"internal_name":"Achievement2","localized_name":"To Greener Pastures","localized_desc":"Complete Chapter 2.","hidden":true}]}}`))
	if len(words) != 2 {
		t.Fatalf("words = %v", words)
	}
	if StoryOf(list) != nil {
		t.Error("a story without words")
	}
	filled := StoryWords(list, words)
	if s := StoryOf(filled); s == nil || labels(s) != "+Chapter 1; Chapter 2;" {
		t.Errorf("filled = %+v", s)
	}
	if list[0].Description != "" || list[1].Name != "Achievement2" {
		t.Error("StoryWords changed the list it was given")
	}
	if ParsePublicAchievements([]byte(`{"response":{}}`)) != nil {
		t.Error("an empty answer gave words")
	}
}
