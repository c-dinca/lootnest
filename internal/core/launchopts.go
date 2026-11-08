package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LaunchOptions are how Lootnest starts a game it starts itself, and what it
// does around each time it's played, stored in Config.LaunchOpts by Library
// key. The game's arguments stay where they were before (Game.Launch.Args,
// LibraryPref.Args).
type LaunchOptions struct {
	// WorkDir and RunAsAdmin are for games started by their exe: the folder
	// the game starts in (its own when empty), and Windows' "Run as
	// administrator".
	WorkDir    string `json:"workDir,omitempty"`
	RunAsAdmin bool   `json:"runAsAdmin,omitempty"`
	// Before the game starts, CloseApps are asked to close and StartBefore
	// start; once it closes, StartAfter start and, with ReopenClosed, the
	// apps that were closed open again.
	CloseApps    []CloseApp `json:"closeApps,omitempty"`
	StartBefore  []Program  `json:"startBefore,omitempty"`
	StartAfter   []Program  `json:"startAfter,omitempty"`
	ReopenClosed bool       `json:"reopenClosed,omitempty"`
}

// CloseApp is a program to close before play, by its process name
// ("Discord.exe"). Force ends it when it doesn't close by itself.
type CloseApp struct {
	Name  string `json:"name"`
	Force bool   `json:"force,omitempty"`
}

// Program is a program to start, with its arguments.
type Program struct {
	Path string `json:"path"`
	Args string `json:"args,omitempty"`
}

// HasSteps says whether anything runs before or after play.
func (o LaunchOptions) HasSteps() bool {
	return len(o.CloseApps) > 0 || len(o.StartBefore) > 0 || len(o.StartAfter) > 0
}

// Moved is the options for a new copy of the game in to: a folder to start
// in inside the old copy's folder from moves to the same place in to, or
// back to the game's own folder when to has none there.
func (o LaunchOptions) Moved(from, to string) LaunchOptions {
	if rel, err := filepath.Rel(from, o.WorkDir); o.WorkDir != "" && err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		o.WorkDir = filepath.Join(to, rel)
		if st, err := os.Stat(o.WorkDir); err != nil || !st.IsDir() {
			o.WorkDir = ""
		}
	}
	return o
}

// Empty says whether the options change nothing.
func (o LaunchOptions) Empty() bool { return o.WorkDir == "" && !o.RunAsAdmin && !o.HasSteps() }

const (
	closeAppsMax = 10
	programsMax  = 5
)

// keepRunning are Windows' own processes and Lootnest: closing them would
// close the desktop, or Lootnest itself.
var keepRunning = map[string]bool{
	"explorer.exe": true, "dwm.exe": true, "csrss.exe": true, "winlogon.exe": true, "wininit.exe": true,
	"services.exe": true, "lsass.exe": true, "smss.exe": true, "svchost.exe": true, "sihost.exe": true,
	"ctfmon.exe": true, "conhost.exe": true, "taskhostw.exe": true, "runtimebroker.exe": true,
	"startmenuexperiencehost.exe": true, "shellexperiencehost.exe": true, "searchhost.exe": true,
	"textinputhost.exe": true, "lootnest.exe": true,
}

// CleanLaunchOptions trims a game's launch options and checks them: the
// folder exists, programs are .exe files that exist, apps to close are
// process names without a path.
func CleanLaunchOptions(o LaunchOptions) (LaunchOptions, error) {
	o.WorkDir = strings.TrimSpace(o.WorkDir)
	if o.WorkDir != "" {
		if !filepath.IsAbs(o.WorkDir) {
			return o, errors.New("Choose the folder to start in with Browse.")
		}
		if st, err := os.Stat(o.WorkDir); err != nil || !st.IsDir() {
			return o, fmt.Errorf("Lootnest can't find the folder %s.", o.WorkDir)
		}
	}
	if len(o.CloseApps) > closeAppsMax {
		return o, fmt.Errorf("Choose up to %d apps to close.", closeAppsMax)
	}
	seen := map[string]bool{}
	apps := []CloseApp{}
	for _, c := range o.CloseApps {
		c.Name = strings.TrimSpace(c.Name)
		if c.Name == "" || len(c.Name) > 100 || strings.ContainsAny(c.Name, `\/:*?"<>|`+"\r\n\t\x00") {
			return o, fmt.Errorf("%q isn't a program's name, such as Discord.exe.", c.Name)
		}
		if !strings.EqualFold(filepath.Ext(c.Name), ".exe") {
			c.Name += ".exe"
		}
		low := strings.ToLower(c.Name)
		if keepRunning[low] {
			return o, fmt.Errorf("%s is part of Windows or Lootnest, so Lootnest doesn't close it.", c.Name)
		}
		if !seen[low] {
			seen[low] = true
			apps = append(apps, c)
		}
	}
	o.CloseApps = apps
	var err error
	if o.StartBefore, err = cleanPrograms(o.StartBefore); err != nil {
		return o, err
	}
	if o.StartAfter, err = cleanPrograms(o.StartAfter); err != nil {
		return o, err
	}
	return o, nil
}

func cleanPrograms(ps []Program) ([]Program, error) {
	if len(ps) > programsMax {
		return nil, fmt.Errorf("Choose up to %d programs to start.", programsMax)
	}
	out := []Program{}
	for _, p := range ps {
		p.Path, p.Args = strings.TrimSpace(p.Path), strings.TrimSpace(p.Args)
		if !filepath.IsAbs(p.Path) || !strings.EqualFold(filepath.Ext(p.Path), ".exe") {
			return nil, fmt.Errorf("Choose a program (.exe) to start, not %q.", p.Path)
		}
		if st, err := os.Stat(p.Path); err != nil || !st.Mode().IsRegular() {
			return nil, fmt.Errorf("Lootnest can't find %s.", p.Path)
		}
		if len(p.Args) > 1000 || strings.ContainsAny(p.Args, "\r\n\t\x00") {
			return nil, errors.New("A program's options must fit on one line.")
		}
		out = append(out, p)
	}
	return out, nil
}
