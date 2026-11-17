package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// A full update: a whole new build of a game, already downloaded and
// unpacked into a staging folder, takes the place of the game's folder.
// The old folder is never changed: it is renamed aside
// (<folder>.lootnest-old) and stays until the user has played the new
// build or asks for it back.
//
// It runs as a state machine. The state is written to disk (temp file,
// sync, rename) before each step's work starts, so a crash or a power cut
// at any point is found again at the next start and put right
// (RecoverFullUpdate):
//
//	staged   the build is ready and the user hasn't said yes yet
//	backup   the saves are being copied
//	swap     the old folder is renamed aside and the build moved in
//	restore  the files to keep are copied from the old folder into the new
//	verify   the game's exe is looked for
//	kept     done; the old folder is kept and Roll back is offered
//	done     the old folder is gone
//	failed   it failed and the old folder is back in place
//	rolledback  the user took the old folder back
// ---------------------------------------------------------------------------

const (
	FUStaged     = "staged"
	FUBackup     = "backup"
	FUSwap       = "swap"
	FURestore    = "restore"
	FUVerify     = "verify"
	FUKept       = "kept"
	FUDone       = "done"
	FUFailed     = "failed"
	FURolledBack = "rolledback"
)

// FullUpdate is one full update of a game, as kept in state.json.
type FullUpdate struct {
	ID      string    `json:"id"`
	Key     string    `json:"key"` // the game's Library key
	Name    string    `json:"name"`
	Dir     string    `json:"dir"`             // the game's folder
	Exe     string    `json:"exe"`             // the game's exe, relative to Dir, before the update
	Title   string    `json:"title,omitempty"` // what the source calls the build
	Version string    `json:"version,omitempty"`
	Source  string    `json:"source,omitempty"`
	Plugin  string    `json:"plugin,omitempty"`
	Offer   string    `json:"offer,omitempty"`
	Staging string    `json:"staging"`         // where the download and the unpacked build are
	Build   string    `json:"build"`           // the new build's root, inside Staging
	NewExe  string    `json:"newExe"`          // the exe in the new build, relative to its root
	Keep    []string  `json:"keep,omitempty"`  // paths, relative to Dir, carried from the old folder
	Old     string    `json:"old,omitempty"`   // where the old folder is kept
	Aside   string    `json:"aside,omitempty"` // a build that failed or was rolled back, for the Recycle Bin
	Saves   string    `json:"saves,omitempty"` // the copy of the saves made first
	State   string    `json:"state"`
	Error   string    `json:"error,omitempty"`
	At      time.Time `json:"at"` // when the state last changed
	Swapped time.Time `json:"swapped,omitempty"`
}

// SaveFullUpdate writes u to dir\state.json: a temp file, synced, renamed
// into place, so the file is always a whole state.
func SaveFullUpdate(dir string, u *FullUpdate) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	u.At = time.Now()
	b, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "state.json.tmp")
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "state.json"))
}

// LoadFullUpdate reads dir\state.json; nil when there is none.
func LoadFullUpdate(dir string) (*FullUpdate, error) {
	b, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var u FullUpdate
	if err := json.Unmarshal(b, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// FullUpdateRefused says why the game in dir can't have its folder
// replaced, or nil.
func FullUpdateRefused(dir string) error {
	clean := filepath.Clean(dir)
	if !filepath.IsAbs(clean) || tooBroadDir(clean) || !isDir(clean) {
		return errors.New("the game's folder isn't one Lootnest can replace")
	}
	for _, p := range strings.FieldsFunc(strings.ToLower(clean), func(r rune) bool { return r == '\\' || r == '/' }) {
		if p == "program files" || p == "program files (x86)" || p == "windowsapps" {
			return errors.New("Windows protects the games under Program Files and WindowsApps, so Lootnest doesn't replace a game's folder there")
		}
	}
	if isDir(clean + fullOldSuffix) {
		return errors.New("the folder from the update before is still kept. Roll that update back, or let Lootnest remove the old folder, first")
	}
	return nil
}

const fullOldSuffix = ".lootnest-old"

// FindBuild looks in a staged package for a full build of the game: a
// folder with the game's exe (exeRel, relative to the game's folder) in the
// same place, or anywhere up to four folders down. It returns the build's
// root and the exe's path inside it. errNoBuild when the package has no
// such exe: it's a patch, an installer or something else.
func FindBuild(staged, exeRel string) (root, newExe string, err error) {
	rel, err := updateRel(exeRel)
	if err != nil {
		return "", "", errors.New("the game's exe isn't inside its folder")
	}
	name := strings.ToLower(filepath.Base(rel))
	var found []string
	base := filepath.Clean(staged)
	_ = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		r, _ := filepath.Rel(base, p)
		depth := strings.Count(filepath.ToSlash(r), "/")
		if d.IsDir() {
			if depth >= 5 && p != base {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && strings.ToLower(d.Name()) == name {
			found = append(found, filepath.ToSlash(r))
		}
		return nil
	})
	if len(found) == 0 {
		return "", "", ErrNoBuild
	}
	sort.Slice(found, func(i, j int) bool {
		if a, b := strings.Count(found[i], "/"), strings.Count(found[j], "/"); a != b {
			return a < b
		}
		return found[i] < found[j]
	})
	// The build's root, by default: the package's, under its single
	// wrapping folders.
	root = base
	for {
		ents, err := os.ReadDir(root)
		if err != nil || len(ents) != 1 || !ents[0].IsDir() {
			break
		}
		root = filepath.Join(root, ents[0].Name())
	}
	relTo := func(f string) string {
		r, err := filepath.Rel(root, filepath.Join(base, filepath.FromSlash(f)))
		if err != nil || strings.HasPrefix(r, "..") {
			return ""
		}
		return filepath.ToSlash(r)
	}
	lr := strings.ToLower(rel)
	// The exe where it was before.
	for _, f := range found {
		if strings.ToLower(relTo(f)) == lr {
			return root, relTo(f), nil
		}
	}
	// The exe under the same folders as before, beside other things in the
	// package: the build starts where those folders do.
	if strings.Contains(rel, "/") {
		for _, f := range found {
			if lf := strings.ToLower(f); strings.HasSuffix(lf, "/"+lr) {
				return filepath.Join(base, filepath.FromSlash(f[:len(f)-len(rel)])), f[len(f)-len(rel):], nil
			}
		}
	}
	// Elsewhere in the build.
	if r := relTo(found[0]); r != "" {
		return root, r, nil
	}
	return "", "", ErrNoBuild
}

// ErrNoBuild: the package doesn't hold the game's exe.
var ErrNoBuild = errors.New("the package doesn't hold the whole game")

// KeepItem is something carried from the old folder into the new build.
type KeepItem struct {
	Rel  string `json:"rel"`
	Why  string `json:"why"` // "saves" or "yours"
	Size int64  `json:"size"`
}

// FullPlan is what a full update would do, for the user to read first.
type FullPlan struct {
	Keep []KeepItem `json:"keep"`
	// Extra are things at the top of the old folder that the new build
	// doesn't have (a mods folder, a settings file): the user may tick
	// them to keep. Files of a store or of a tool that stands in for one
	// are never offered.
	Extra     []KeepItem `json:"extra"`
	BuildSize int64      `json:"buildSize"`
	OldSize   int64      `json:"oldSize"`
}

// keepRel checks a path to keep: inside the folder, and not a file of a
// store or of a tool that stands in for one.
func keepRel(rel string) (string, error) {
	r, err := updateRel(rel)
	if err != nil {
		return "", errors.New("a path to keep must be inside the game's folder (" + rel + ")")
	}
	if keepBlocked(r) {
		return "", errors.New(r + " belongs to a store or to a tool that stands in for one, so Lootnest leaves it to the new build")
	}
	return r, nil
}

// keepBlocked: files of a store or of a tool that stands in for one (the
// mods' list), and such a tool's settings folder itself.
func keepBlocked(rel string) bool {
	if modBlockedFile(rel) {
		return true
	}
	for _, p := range strings.Split(strings.ToLower(rel), "/") {
		if p == "steam_settings" {
			return true
		}
	}
	return false
}

// PlanFullUpdate lists what would be kept from dir when build replaces
// it: the save locations inside the folder (saves, relative to dir) and
// the paths the user chose (yours), where they exist; and what else only
// the old folder has at its top.
func PlanFullUpdate(dir, build string, saves, yours []string) (*FullPlan, error) {
	p := &FullPlan{Keep: []KeepItem{}, Extra: []KeepItem{}, BuildSize: Size(build), OldSize: Size(dir)}
	seen := map[string]bool{}
	add := func(list []string, why string) error {
		for _, rel := range list {
			r, err := keepRel(rel)
			if err != nil {
				if why == "saves" {
					continue // a save location that can't be carried is still in the copy made first
				}
				return err
			}
			if seen[strings.ToLower(r)] {
				continue
			}
			full := filepath.Join(dir, filepath.FromSlash(r))
			if _, err := os.Lstat(full); err != nil {
				continue
			}
			seen[strings.ToLower(r)] = true
			p.Keep = append(p.Keep, KeepItem{Rel: r, Why: why, Size: Size(full)})
		}
		return nil
	}
	if err := add(saves, "saves"); err != nil {
		return nil, err
	}
	if err := add(yours, "yours"); err != nil {
		return nil, err
	}
	ents, _ := os.ReadDir(dir)
	for _, e := range ents {
		name := e.Name()
		if seen[strings.ToLower(name)] || keepBlocked(name) || e.Type()&fs.ModeSymlink != 0 {
			continue
		}
		if _, err := os.Lstat(filepath.Join(build, name)); err == nil {
			continue
		}
		covered := false
		for k := range seen {
			covered = covered || strings.HasPrefix(k, strings.ToLower(name)+"/")
		}
		if !covered {
			p.Extra = append(p.Extra, KeepItem{Rel: name, Why: "yours", Size: Size(filepath.Join(dir, name))})
		}
	}
	return p, nil
}

// FullUpdateHooks is what the app does around the file work.
type FullUpdateHooks struct {
	Save   func(*FullUpdate) error // writes the state
	Log    func(string)
	Backup func() (id string, err error) // copies the game's saves; called before anything moves
	// RestoreSaves puts the copy made by Backup back where an update that
	// failed changed the saves; nil when there's nothing to do.
	RestoreSaves func(id string) error
	// After is for tests: called after each state's work, it can fail.
	After func(state string) error
}

func (h FullUpdateHooks) log(format string, args ...any) {
	if h.Log != nil {
		h.Log(fmt.Sprintf(format, args...))
	}
}

func (h FullUpdateHooks) to(u *FullUpdate, state string) error {
	u.State = state
	if h.Save == nil {
		return nil
	}
	return h.Save(u)
}

func (h FullUpdateHooks) after(state string) error {
	if h.After == nil {
		return nil
	}
	return h.After(state)
}

// RunFullUpdate takes a staged update through to "kept". On any failure
// after the swap began it puts the old folder back and returns the error;
// u.State is then "failed".
func RunFullUpdate(u *FullUpdate, h FullUpdateHooks) error {
	if u.State != FUStaged {
		return errors.New("that update isn't waiting to be applied")
	}
	if err := FullUpdateRefused(u.Dir); err != nil {
		return err
	}
	if !isDir(u.Build) || !isFile(filepath.Join(u.Build, filepath.FromSlash(u.NewExe))) {
		return errors.New("the new build isn't where it was unpacked any more")
	}
	keep := make([]string, 0, len(u.Keep))
	for _, rel := range u.Keep {
		r, err := keepRel(rel)
		if err != nil {
			return err
		}
		keep = append(keep, r)
	}
	u.Keep = keep

	// 1. The saves, before anything moves.
	if err := h.to(u, FUBackup); err != nil {
		return err
	}
	h.log("Copying the saves first")
	if h.Backup != nil {
		id, err := h.Backup()
		if err != nil {
			u.Error = "the saves couldn't be copied first: " + err.Error()
			_ = h.to(u, FUFailed)
			return errors.New(u.Error)
		}
		u.Saves = id
		h.log("Saves copied (%s)", id)
	}
	if err := h.after(FUBackup); err != nil {
		u.Error = err.Error()
		_ = h.to(u, FUFailed)
		return err
	}

	// 2. The swap. From here a failure puts the old folder back.
	u.Old = u.Dir + fullOldSuffix
	if err := h.to(u, FUSwap); err != nil {
		return err
	}
	fail := func(step string, err error) error {
		h.log("%s failed: %v", step, err)
		u.Error = step + " failed: " + err.Error()
		if rerr := undoSwap(u, h); rerr != nil {
			u.Error += ". Putting the old folder back failed too: " + rerr.Error()
			_ = h.to(u, FUFailed)
			return errors.New(u.Error)
		}
		_ = h.to(u, FUFailed)
		return errors.New(u.Error)
	}
	h.log("Moving %s aside to %s", u.Dir, u.Old)
	if err := os.Rename(u.Dir, u.Old); err != nil {
		u.Old = ""
		u.Error = "the game's folder couldn't be moved aside (a file in it may be open): " + err.Error()
		_ = h.to(u, FUFailed)
		return errors.New(u.Error)
	}
	h.log("Moving the new build into place")
	if err := moveDir(u.Build, u.Dir); err != nil {
		return fail("moving the new build into place", err)
	}
	u.Swapped = time.Now()
	if err := h.after(FUSwap); err != nil {
		return fail("the swap", err)
	}

	// 3. What's kept from the old folder.
	if err := h.to(u, FURestore); err != nil {
		return fail("writing the state", err)
	}
	for _, rel := range u.Keep {
		src, dst := filepath.Join(u.Old, filepath.FromSlash(rel)), filepath.Join(u.Dir, filepath.FromSlash(rel))
		info, err := os.Lstat(src)
		if err != nil {
			continue // not there any more: nothing to carry
		}
		h.log("Keeping %s", rel)
		if info.IsDir() {
			err = copyTree(src, dst)
		} else {
			if err = os.MkdirAll(filepath.Dir(dst), 0o755); err == nil {
				err = copyFile(src, dst, info)
			}
		}
		if err != nil {
			return fail("keeping "+rel, err)
		}
	}
	if err := h.after(FURestore); err != nil {
		return fail("keeping your files", err)
	}

	// 4. Is it a game?
	if err := h.to(u, FUVerify); err != nil {
		return fail("writing the state", err)
	}
	if !isFile(filepath.Join(u.Dir, filepath.FromSlash(u.NewExe))) {
		return fail("checking the new build", errors.New("its exe, "+u.NewExe+", isn't there"))
	}
	if err := h.after(FUVerify); err != nil {
		return fail("checking the new build", err)
	}
	h.log("Done. The old folder is kept at %s", u.Old)
	u.Error = ""
	return h.to(u, FUKept)
}

// undoSwap puts the old folder back where the game was. Whatever is at the
// game's path (a build half moved in) is moved aside first, never deleted.
func undoSwap(u *FullUpdate, h FullUpdateHooks) error {
	if u.Old == "" || !isDir(u.Old) {
		if isDir(u.Dir) {
			return nil // the swap never began
		}
		return errors.New("neither the game's folder nor its old copy is there")
	}
	if _, err := os.Lstat(u.Dir); err == nil {
		aside := freePath(u.Dir + ".lootnest-failed")
		h.log("Moving what's at %s aside to %s", u.Dir, aside)
		if err := os.Rename(u.Dir, aside); err != nil {
			return err
		}
		u.Aside = aside
	}
	h.log("Putting the old folder back")
	if err := os.Rename(u.Old, u.Dir); err != nil {
		return err
	}
	u.Old = ""
	if h.RestoreSaves != nil && u.Saves != "" {
		if err := h.RestoreSaves(u.Saves); err != nil {
			h.log("The saves' copy couldn't be put back: %v", err)
			return errors.New("the game's folder is back, but its saves couldn't be put back from the copy: " + err.Error())
		}
	}
	return nil
}

// freePath is p, or p with a number after it when p is taken.
func freePath(p string) string {
	if _, err := os.Lstat(p); err != nil {
		return p
	}
	for i := 2; ; i++ {
		q := fmt.Sprintf("%s-%d", p, i)
		if _, err := os.Lstat(q); err != nil {
			return q
		}
	}
}

// RecoverFullUpdate puts right an update that was cut short (Lootnest
// closed, a crash, a power cut), at the next start. An update that hadn't
// begun to swap goes back to waiting; one caught in or after the swap gets
// the old folder back. It says whether it changed anything.
func RecoverFullUpdate(u *FullUpdate, h FullUpdateHooks) (bool, error) {
	switch u.State {
	case FUBackup:
		// Nothing moved yet.
		u.Error = ""
		return true, h.to(u, FUStaged)
	case FUSwap, FURestore, FUVerify:
		if u.Old == "" {
			u.Old = u.Dir + fullOldSuffix
		}
		h.log("An update of %s was cut short while %s", u.Name, map[string]string{FUSwap: "its folder was being replaced", FURestore: "your files were being carried over", FUVerify: "the new build was being checked"}[u.State])
		was := u.State
		if err := undoSwap(u, h); err != nil {
			u.Error = "an update was cut short, and the old folder couldn't be put back: " + err.Error()
			_ = h.to(u, FUFailed)
			return true, errors.New(u.Error)
		}
		u.Error = "The update was cut short (" + was + "). The game is back as it was."
		return true, h.to(u, FUFailed)
	}
	return false, nil
}

// RollbackFullUpdate takes the old folder back for an update that's
// "kept": the saves are copied first (they may be newer than the copy made
// before the update), the new build is moved aside (u.Aside, for the
// Recycle Bin) and the old folder renamed back. The saves outside the
// folder stay as they are.
func RollbackFullUpdate(u *FullUpdate, h FullUpdateHooks) error {
	if u.State != FUKept || u.Old == "" || !isDir(u.Old) {
		return errors.New("the folder from before that update isn't kept any more")
	}
	if h.Backup != nil {
		if _, err := h.Backup(); err != nil {
			return errors.New("nothing was changed: the saves couldn't be copied first (" + err.Error() + ")")
		}
	}
	aside := freePath(u.Dir + ".lootnest-undone")
	if err := os.Rename(u.Dir, aside); err != nil {
		return errors.New("the game's folder couldn't be moved aside (a file in it may be open): " + err.Error())
	}
	if err := os.Rename(u.Old, u.Dir); err != nil {
		_ = os.Rename(aside, u.Dir)
		return err
	}
	u.Aside, u.Old = aside, ""
	h.log("Rolled back: the old folder is in place, the new build is at %s", aside)
	return h.to(u, FURolledBack)
}

// FinishFullUpdate ends the keeping of the old folder: it returns its path
// for the Recycle Bin and marks the update done.
func FinishFullUpdate(u *FullUpdate, h FullUpdateHooks) (old string, err error) {
	if u.State != FUKept {
		return "", errors.New("that update has nothing kept")
	}
	old, u.Old = u.Old, ""
	return old, h.to(u, FUDone)
}

// moveDir moves a folder: a rename where it can be one (the same drive),
// else a copy that's compared with the original before the original goes.
func moveDir(src, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return errors.New(dst + " is in the way")
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyTree(src, dst); err != nil {
		return err
	}
	a, err := SnapshotDir(src)
	if err != nil {
		return err
	}
	b, err := SnapshotDir(dst)
	if err != nil {
		return err
	}
	if len(a) != len(b) {
		return errors.New("the copy doesn't have every file")
	}
	for rel, st := range a {
		if b[rel].Size != st.Size {
			return errors.New("the copy of " + rel + " differs")
		}
	}
	return os.RemoveAll(src)
}
