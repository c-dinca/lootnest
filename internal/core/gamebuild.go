package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// ---------------------------------------------------------------------------
// Which copy of a game is on the PC, to tell when it updated since it last
// started: Steam's build id for a Steam game, else its exe's size and time.
// builds.json keeps the last one seen per Library key, so the first start
// after an update can make a save point before the new version touches the
// saves.
// ---------------------------------------------------------------------------

// GameBuild names the copy of a game on the PC, or "" when it can't tell.
// With a Steam app id it reads the build id in Steam's manifest, or the
// build an update Steam has waiting will bring, since Steam may only apply
// it once asked to start the game; otherwise it stamps the exe.
func GameBuild(steamPath string, appID int, exe string) string {
	if appID > 0 {
		for _, lib := range SteamLibraries(steamPath) {
			b, err := os.ReadFile(filepath.Join(lib, "steamapps", "appmanifest_"+strconv.Itoa(appID)+".acf"))
			if err != nil {
				continue
			}
			if m, err := ParseVDF(string(b)); err == nil {
				st := vdfMap(m, "AppState")
				id := vdfStr(st, "buildid")
				if t := vdfStr(st, "TargetBuildID"); t != "" && t != "0" {
					id = t
				}
				if id != "" && id != "0" {
					return "build:" + id
				}
			}
		}
		return ""
	}
	if exe == "" {
		return ""
	}
	st, err := os.Stat(exe)
	if err != nil || st.IsDir() {
		return ""
	}
	return "exe:" + strconv.FormatInt(st.Size(), 10) + ":" + strconv.FormatInt(st.ModTime().Unix(), 10)
}

func (s *Store) BuildsPath() string { return filepath.Join(s.Dir, "builds.json") }

// NoteBuild records the build of a game about to start and says whether it
// differs from the one seen last time. The first time a game is seen, and
// when its build isn't known, it's false.
func (s *Store) NoteBuild(key, build string) (bool, error) {
	if build == "" {
		return false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m := map[string]string{}
	if b, err := os.ReadFile(s.BuildsPath()); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	old := m[key]
	if old == build {
		return false, nil
	}
	m[key] = build
	return old != "", writeJSONAtomic(s.BuildsPath(), m)
}
