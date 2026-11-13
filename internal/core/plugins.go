package core

import (
	"archive/zip"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Plugins, phase 1: data only. A plugin is a .zip (or, while it's being
// made, a folder) with plugin.json at its top and nothing but data files.
// It can add games to the Library (sources/*.json), suggest save locations
// (saves/*.yaml, Ludusavi's format), fill in game details (metadata/*.json)
// and change colours (theme.json). Nothing in it is ever run or loaded as
// code, and nothing in it can write saves or game folders, start programs
// or go online.
//
// Installed plugins live in plugins\<id>; a replaced version is kept in
// plugins\.old\<id>\<version> for Roll back; plugins.json says which are on.
// ---------------------------------------------------------------------------

const (
	pluginFileMax  = 8 << 20
	pluginTotalMax = 32 << 20
	pluginFilesMax = 500
)

// PluginKinds are what a plugin can provide.
var PluginKinds = []string{"sources", "saves", "metadata", "theme"}

var (
	pluginIDRe      = regexp.MustCompile(`^[a-z0-9-]{3,40}$`)
	pluginVersionRe = regexp.MustCompile(`^\d{1,6}\.\d{1,6}\.\d{1,6}$`)
	pluginExts      = []string{".json", ".ini", ".vdf", ".yaml", ".yml", ".txt", ".png", ".jpg", ".webp"}
)

type PluginManifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	Provides    []string `json:"provides"`
}

// Plugin is an installed plugin as Settings lists it. Problem says why a
// plugin that no longer checks out is left off.
type Plugin struct {
	PluginManifest
	On      bool   `json:"on"`
	Older   string `json:"older,omitempty"` // the version Roll back brings back
	Key     string `json:"key,omitempty"`   // the fingerprint of the key it's signed with
	Problem string `json:"problem,omitempty"`
	Dir     string `json:"-"`
}

func (p Plugin) Has(kind string) bool { return slices.Contains(p.Provides, kind) }

func PluginsDir(dataDir string) string { return filepath.Join(dataDir, "plugins") }

func pluginsState(dataDir string) string { return filepath.Join(dataDir, "plugins.json") }

func pluginOld(dataDir, id string) string { return filepath.Join(PluginsDir(dataDir), ".old", id) }

// checkPluginFile is the rule for one file inside a plugin.
func checkPluginFile(name string, size int64) (string, error) {
	rel, err := updateRel(name)
	if err != nil {
		return "", errors.New("it has a file outside its own folder (" + name + ")")
	}
	for _, part := range strings.Split(rel, "/") {
		if !safeComponent(part) {
			return "", errors.New("it has a file Windows can't name (" + rel + ")")
		}
	}
	if !slices.Contains(pluginExts, strings.ToLower(filepath.Ext(rel))) {
		return "", errors.New("it has a file that isn't data (" + rel + "); plugins can only hold .json, .ini, .vdf, .yaml, .txt and pictures")
	}
	if size > pluginFileMax {
		return "", errors.New(rel + " is over 8 MB")
	}
	return rel, nil
}

// pluginTally counts files and bytes against the limits.
type pluginTally struct {
	files int
	bytes int64
}

func (t *pluginTally) add(size int64) error {
	t.files++
	t.bytes += size
	if t.files > pluginFilesMax {
		return errors.New("it has more than 500 files")
	}
	if t.bytes > pluginTotalMax {
		return errors.New("it holds more than 32 MB")
	}
	return nil
}

// CheckPluginDir checks every file in a plugin's folder and what it
// provides, and returns its manifest.
func CheckPluginDir(dir string) (*PluginManifest, error) {
	var t pluginTally
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == dir {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			return errors.New("it has a link (" + filepath.ToSlash(rel) + ")")
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if _, err := checkPluginFile(filepath.ToSlash(rel), info.Size()); err != nil {
			return err
		}
		return t.add(info.Size())
	})
	if err != nil {
		return nil, err
	}
	m, err := readPluginManifest(dir)
	if err != nil {
		return nil, err
	}
	for _, kind := range m.Provides {
		if err := checkPluginKind(dir, kind); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func readPluginManifest(dir string) (*PluginManifest, error) {
	b, err := os.ReadFile(filepath.Join(dir, "plugin.json"))
	if err != nil {
		return nil, errors.New("plugin.json isn't at its top")
	}
	var m PluginManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, errors.New("plugin.json can't be read: " + shortErr(err))
	}
	for _, s := range []*string{&m.Name, &m.Author, &m.Description} {
		*s = strings.Join(strings.Fields(*s), " ")
	}
	switch {
	case !pluginIDRe.MatchString(m.ID):
		return nil, errors.New("its id must be 3 to 40 lowercase letters, digits or hyphens")
	case strings.HasPrefix(m.ID, "ln-"):
		return nil, errors.New(`ids starting "ln-" are kept for Lootnest's own themes`)
	case m.Name == "" || len([]rune(m.Name)) > 60:
		return nil, errors.New("its name must be 1 to 60 characters")
	case !pluginVersionRe.MatchString(m.Version):
		return nil, errors.New("its version must look like 1.0.0")
	case len([]rune(m.Author)) > 60:
		return nil, errors.New("its author must be under 60 characters")
	case len([]rune(m.Description)) > 300:
		return nil, errors.New("its description must be under 300 characters")
	case len(m.Provides) == 0:
		return nil, errors.New("it doesn't say what it provides")
	}
	seen := map[string]bool{}
	for _, k := range m.Provides {
		if !slices.Contains(PluginKinds, k) || seen[k] {
			return nil, fmt.Errorf("it provides %q, which isn't sources, saves, metadata or theme", k)
		}
		seen[k] = true
	}
	return &m, nil
}

// checkPluginKind reads what a plugin says it provides, so a plugin that
// wouldn't work is refused at once, with the reason.
func checkPluginKind(dir, kind string) error {
	var err error
	switch kind {
	case "sources":
		_, err = PluginSources(dir)
	case "saves":
		_, err = PluginSaves(dir)
	case "metadata":
		_, err = PluginMetadata(dir)
	case "theme":
		_, err = PluginTheme(dir)
	}
	return err
}

// pluginFiles are a plugin's files of one kind: dir\sub\*.ext.
func pluginFiles(dir, sub string, exts ...string) []string {
	ents, _ := os.ReadDir(filepath.Join(dir, sub))
	var out []string
	for _, e := range ents {
		if !e.IsDir() && slices.Contains(exts, strings.ToLower(filepath.Ext(e.Name()))) {
			out = append(out, filepath.Join(dir, sub, e.Name()))
		}
	}
	return out
}

// PluginInstalled is what an install did. PinnedNow is a signed version
// replacing an unsigned install: its key is pinned from now on.
type PluginInstalled struct {
	PluginManifest
	Key       string `json:"key,omitempty"`
	PinnedNow bool   `json:"pinnedNow,omitempty"`
}

// InstallPlugin installs a plugin from a .zip or a folder. It's checked
// in full in a folder of its own first, its signature too; a plugin with
// the same id is replaced only then, and kept in plugins\.old for Roll
// back. A version signed with another key than the pinned one is refused
// with a *PluginKeyChange.
func InstallPlugin(dataDir, src string) (*PluginInstalled, error) {
	return installPlugin(dataDir, src, "")
}

// InstallPluginNewKey installs a version signed with another key than the
// pinned one, and pins the new key, once the user accepted it: accepted is
// the fingerprint they saw. A version signed with any other key is refused
// with a *PluginKeyChange again.
func InstallPluginNewKey(dataDir, src, accepted string) (*PluginInstalled, error) {
	return installPlugin(dataDir, src, accepted)
}

func installPlugin(dataDir, src, accepted string) (*PluginInstalled, error) {
	root := PluginsDir(dataDir)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp(root, ".new-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	st, err := os.Stat(src)
	switch {
	case err != nil:
		return nil, errors.New("the plugin isn't there")
	case st.IsDir():
		if _, err := CheckPluginDir(src); err != nil {
			return nil, err
		}
		err = copyTree(src, tmp)
	case strings.EqualFold(filepath.Ext(src), ".zip"):
		if err = checkPluginZip(src); err == nil {
			_, err = StageUpdateMax(src, tmp, pluginTotalMax)
		}
	default:
		return nil, errors.New("a plugin is a .zip")
	}
	if err != nil {
		return nil, err
	}
	m, err := CheckPluginDir(tmp)
	if err != nil {
		return nil, err
	}
	key, err := pluginKey(tmp)
	if err != nil {
		return nil, err
	}
	state := loadPlugins(dataDir)
	var kc *PluginKeyChange
	if err := checkPluginPin(m, state.Keys[m.ID], key, "this version is"); err != nil && !(errors.As(err, &kc) && accepted != "" && kc.Key == accepted) {
		return nil, err
	}
	dst := filepath.Join(root, m.ID)
	out := &PluginInstalled{PluginManifest: *m}
	if key != nil {
		out.Key = KeyFingerprint(key)
		out.PinnedNow = state.Keys[m.ID] == "" && isDir(dst)
	}
	var kept string
	if isDir(dst) {
		old := "0.0.0"
		if om, err := readPluginManifest(dst); err == nil {
			old = om.Version
		}
		kept = filepath.Join(pluginOld(dataDir, m.ID), old)
		_ = os.RemoveAll(kept)
		if err := os.MkdirAll(filepath.Dir(kept), 0o755); err != nil {
			return nil, err
		}
		if err := os.Rename(dst, kept); err != nil {
			return nil, err
		}
	}
	if err := os.Rename(tmp, dst); err != nil {
		if kept != "" {
			_ = os.Rename(kept, dst)
		}
		return nil, err
	}
	if _, ok := state.On[m.ID]; !ok {
		state.On[m.ID] = true
	}
	if key != nil {
		state.Keys[m.ID] = base64.StdEncoding.EncodeToString(key)
	}
	return out, writeJSONAtomic(pluginsState(dataDir), state)
}

// checkPluginZip checks a .zip's list of files before anything in it is
// extracted.
func checkPluginZip(src string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return errors.New("that .zip can't be opened; it may be damaged")
	}
	defer zr.Close()
	var t pluginTally
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Mode()&fs.ModeSymlink != 0 {
			return errors.New("it has a link (" + f.Name + ")")
		}
		size := int64(min(f.UncompressedSize64, 1<<40))
		if _, err := checkPluginFile(f.Name, size); err != nil {
			return err
		}
		if err := t.add(size); err != nil {
			return err
		}
	}
	return nil
}

// pluginsFile is plugins.json: which plugins are on, and the key pinned
// for each signed one (base64).
type pluginsFile struct {
	On   map[string]bool   `json:"on"`
	Keys map[string]string `json:"keys,omitempty"`
}

func loadPlugins(dataDir string) pluginsFile {
	var f pluginsFile
	if b, err := os.ReadFile(pluginsState(dataDir)); err == nil {
		_ = json.Unmarshal(b, &f)
	}
	if f.On == nil {
		f.On = map[string]bool{}
	}
	if f.Keys == nil {
		f.Keys = map[string]string{}
	}
	return f
}

// SetPluginOn turns an installed plugin on or off.
func SetPluginOn(dataDir, id string, on bool) error {
	if !pluginIDRe.MatchString(id) || !isDir(filepath.Join(PluginsDir(dataDir), id)) {
		return errors.New("that plugin isn't installed")
	}
	st := loadPlugins(dataDir)
	st.On[id] = on
	return writeJSONAtomic(pluginsState(dataDir), st)
}

// ListPlugins is every installed plugin, by name.
func ListPlugins(dataDir string) []Plugin {
	ents, _ := os.ReadDir(PluginsDir(dataDir))
	st := loadPlugins(dataDir)
	var out []Plugin
	for _, e := range ents {
		if !e.IsDir() || !pluginIDRe.MatchString(e.Name()) {
			continue
		}
		dir := filepath.Join(PluginsDir(dataDir), e.Name())
		p := Plugin{Dir: dir}
		m, err := CheckPluginDir(dir)
		var key ed25519.PublicKey
		if err == nil {
			if key, err = pluginKey(dir); err == nil && key == nil && st.Keys[m.ID] != "" {
				pin, _ := base64.StdEncoding.DecodeString(st.Keys[m.ID])
				err = fmt.Errorf("its signature is gone, and the version you installed was signed with key %s", KeyFingerprint(pin))
			} else if err == nil {
				err = checkPluginPin(m, st.Keys[m.ID], key, "its files are")
			}
		}
		if err != nil || m.ID != e.Name() {
			p.ID, p.Name = e.Name(), e.Name()
			p.Problem = "It no longer checks out, so it's off"
			if err != nil {
				p.Problem += ": " + err.Error()
			}
		} else {
			p.PluginManifest, p.On = *m, st.On[m.ID]
			if key != nil {
				p.Key = KeyFingerprint(key)
			}
		}
		p.Older = olderPlugin(dataDir, p.ID, p.Version, st.Keys[p.ID])
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

// ActivePlugins are the plugins that are on and check out.
func ActivePlugins(dataDir string) []Plugin {
	var out []Plugin
	for _, p := range ListPlugins(dataDir) {
		if p.On && p.Problem == "" {
			out = append(out, p)
		}
	}
	return out
}

// olderPlugin is the newest version kept in plugins\.old that's older than
// the one installed and signed as the pin says ("" for none).
func olderPlugin(dataDir, id, current, pinned string) string {
	ents, _ := os.ReadDir(pluginOld(dataDir, id))
	best := ""
	for _, e := range ents {
		v := e.Name()
		if e.IsDir() && pluginVersionRe.MatchString(v) && (current == "" || CompareVersions(v, current) < 0) && (best == "" || CompareVersions(v, best) > 0) {
			if key, err := pluginKey(filepath.Join(pluginOld(dataDir, id), v)); err == nil && checkPluginPin(&PluginManifest{ID: id}, pinned, key, "") == nil {
				best = v
			}
		}
	}
	return best
}

// RollbackPlugin puts back the version kept before the last replace, and
// keeps the current one in plugins\.old\<id>\<version>.
func RollbackPlugin(dataDir, id string) (*PluginManifest, error) {
	if !pluginIDRe.MatchString(id) {
		return nil, errors.New("that plugin isn't installed")
	}
	dst := filepath.Join(PluginsDir(dataDir), id)
	cur := ""
	if m, err := readPluginManifest(dst); err == nil {
		cur = m.Version
	}
	st := loadPlugins(dataDir)
	v := olderPlugin(dataDir, id, cur, st.Keys[id])
	if v == "" {
		return nil, errors.New("no older version of it is kept")
	}
	kept := filepath.Join(pluginOld(dataDir, id), v)
	m, err := CheckPluginDir(kept)
	if err != nil {
		return nil, fmt.Errorf("the version kept no longer checks out: %w", err)
	}
	if m.ID != id {
		return nil, errors.New("the version kept is another plugin")
	}
	key, err := pluginKey(kept)
	if err != nil {
		return nil, fmt.Errorf("the version kept no longer checks out: %w", err)
	}
	// The current version is kept in its place, so a roll back can be undone.
	aside := ""
	if isDir(dst) {
		name := cur
		if name == "" {
			name = "unreadable"
		}
		aside = filepath.Join(pluginOld(dataDir, id), name)
		_ = os.RemoveAll(aside)
		if err := os.Rename(dst, aside); err != nil {
			return nil, err
		}
	}
	if err := os.Rename(kept, dst); err != nil {
		if aside != "" {
			_ = os.Rename(aside, dst)
		}
		return nil, err
	}
	if key != nil && st.Keys[id] == "" {
		st.Keys[id] = base64.StdEncoding.EncodeToString(key)
		return m, writeJSONAtomic(pluginsState(dataDir), st)
	}
	return m, nil
}

// RemovePlugin deletes plugins\<id> and forgets its pinned key, so the
// next install of that id is a first one again.
func RemovePlugin(dataDir, id string) error {
	if !pluginIDRe.MatchString(id) {
		return errors.New("that plugin isn't installed")
	}
	if err := os.RemoveAll(filepath.Join(PluginsDir(dataDir), id)); err != nil {
		return err
	}
	st := loadPlugins(dataDir)
	if _, ok := st.Keys[id]; !ok {
		return nil
	}
	delete(st.Keys, id)
	return writeJSONAtomic(pluginsState(dataDir), st)
}
