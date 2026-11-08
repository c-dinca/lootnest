package core

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf16"
)

// ---------------------------------------------------------------------------
// Games from the other launchers, so the Library has every game on the PC:
//
//   - Ubisoft Connect, Riot, Battle.net, EA app, Amazon Games and Rockstar
//     from Windows' list of installed programs (the uninstall entries each
//     launcher writes for its games);
//   - Xbox and PC Game Pass from the XboxGames folders (MicrosoftGame.config);
//   - itch.io from the itch app's apps folder.
//
// A game a launcher must start gets a launch URI (see SafeLaunchURI);
// the others start from their exe, like any game.
// ---------------------------------------------------------------------------

// UninstallEntry is one program from Windows' installed programs list
// (HKLM/HKCU ...\CurrentVersion\Uninstall\<Key>).
type UninstallEntry struct {
	Key             string
	DisplayName     string
	Publisher       string
	InstallLocation string
	UninstallString string
	DisplayIcon     string
}

var (
	uplayKey      = regexp.MustCompile(`(?i)^Uplay Install (\d+)$`)
	riotProduct   = regexp.MustCompile(`--uninstall-product=([A-Za-z0-9_]+)`)
	riotPatchline = regexp.MustCompile(`--uninstall-patchline=([A-Za-z0-9_]+)`)
	blizzardUID   = regexp.MustCompile(`--uid=([A-Za-z0-9_]+)`)
	amazonID      = regexp.MustCompile(`(?i)Amazon Game Remover\.exe"?\s.*-p\s+"?([A-Za-z0-9-]+)`)
)

// Battle.net's own codes for its games' launch links, by the uid in their
// uninstall entry. A game not listed here starts from its exe.
var battleNetCodes = map[string]string{
	"prometheus": "Pro", "wow": "WoW", "wow_classic": "WoW", "d3": "D3", "fenris": "Fen", "diablo4": "Fen",
	"hs_beta": "WTCG", "heroes": "Hero", "s2": "S2", "s1": "S1", "w3": "W3", "osi": "OSI", "rtro": "RTRO",
	"odin": "ODIN", "viper": "VIPR", "zeus": "ZEUS", "lazarus": "LAZR", "fore": "FORE", "auks": "AUKS", "anbs": "ANBS",
}

// launcherApps are the launchers and store apps themselves, never games.
var launcherApps = []string{"battle.net", "blizzard battle.net", "blizzard app", "ubisoft connect", "uplay", "ea app", "ea desktop", "origin",
	"riot client", "riot vanguard", "rockstar games launcher", "rockstar games social club", "social club", "amazon games",
	"epic games launcher", "epic online services", "gog galaxy", "itch", "xbox", "steam", "playnite", "lootnest", "playswap", "hydra", "hydralauncher"}

// IsLauncherApp says whether a program's name is a launcher's, not a game's:
// the name itself, or with "beta", "launcher", "client" or a note in
// brackets after it.
func IsLauncherApp(name string) bool { return launcherOf(name) != "" }

// launcherOf is the launcher (from launcherApps) a program's name is, or "".
func launcherOf(name string) string {
	n := strings.Join(strings.Fields(strings.ToLower(name)), " ")
	for _, l := range launcherApps {
		if n == l {
			return l
		}
		if rest, ok := strings.CutPrefix(n, l+" "); ok {
			switch {
			case strings.HasPrefix(rest, "("), rest == "beta", rest == "launcher", rest == "app", rest == "client", rest == "desktop", rest == "helper":
				return l
			}
		}
	}
	return ""
}

// tooBroadDir: an install folder that holds many programs, such as a drive
// or Program Files, is never one game.
func tooBroadDir(dir string) bool {
	if filepath.Dir(dir) == dir || len(dir) <= 3 {
		return true
	}
	switch strings.ToLower(filepath.Base(dir)) {
	case "program files", "program files (x86)", "programdata", "windows", "users", "appdata", "local", "roaming", "games", "steamapps", "common":
		return true
	}
	return false
}

// LauncherGames picks the games of known launchers out of the installed
// programs.
func LauncherGames(entries []UninstallEntry) []FoundGame {
	var out []FoundGame
	for _, e := range entries {
		name := strings.TrimSpace(e.DisplayName)
		dir := filepath.Clean(strings.Trim(strings.TrimSpace(e.InstallLocation), `"`))
		if name == "" || e.InstallLocation == "" || IsLauncherApp(name) || tooBroadDir(dir) || !isDir(dir) {
			continue
		}
		un := e.UninstallString
		f := FoundGame{Key: "dir:" + strings.ToLower(dir), Name: name, Dir: dir}
		switch {
		case uplayKey.MatchString(e.Key):
			f.Source, f.URI = "ubisoft", "uplay://launch/"+uplayKey.FindStringSubmatch(e.Key)[1]+"/0"
		case strings.Contains(strings.ToLower(un), "riotclientservices.exe") && riotProduct.MatchString(un):
			product := riotProduct.FindStringSubmatch(un)[1]
			if strings.Contains(strings.ToLower(product), "riot_client") {
				continue
			}
			patch := "live"
			if m := riotPatchline.FindStringSubmatch(un); m != nil {
				patch = m[1]
			}
			client := firstArg(un)
			if !isFile(client) {
				continue
			}
			f.Source, f.Exe, f.Args = "riot", client, "--launch-product="+product+" --launch-patchline="+patch
		case strings.Contains(strings.ToLower(un), "blizzard uninstaller") && blizzardUID.MatchString(un):
			uid := strings.ToLower(blizzardUID.FindStringSubmatch(un)[1])
			if strings.HasPrefix(uid, "battle.net") || strings.HasPrefix(uid, "bna") || uid == "agent" {
				continue
			}
			f.Source = "battlenet"
			if code := battleNetCodes[uid]; code != "" {
				f.URI = "battlenet://" + code
			}
		case amazonID.MatchString(un):
			f.Source, f.URI = "amazon", "amazon-games://play/"+amazonID.FindStringSubmatch(un)[1]
		case isEA(e):
			if l := strings.ToLower(name); l == "ea app" || l == "origin" || strings.HasPrefix(l, "ea desktop") {
				continue
			}
			f.Source = "ea"
		case strings.EqualFold(strings.TrimSpace(e.Publisher), "Rockstar Games"):
			if strings.Contains(strings.ToLower(name), "launcher") || strings.Contains(strings.ToLower(name), "social club") {
				continue
			}
			f.Source = "rockstar"
		default:
			continue
		}
		if f.Exe == "" {
			f.Exe, f.ProcessName = MainExe(dir)
			if f.Exe == "" && f.URI == "" {
				continue // nothing to start it with
			}
		} else {
			_, f.ProcessName = MainExe(dir)
		}
		out = append(out, f)
	}
	return out
}

func isEA(e UninstallEntry) bool {
	un := strings.ToLower(e.UninstallString)
	return strings.Contains(un, "eainstaller") || strings.Contains(un, `\ea desktop\`) || strings.Contains(un, "origin") && strings.Contains(un, "uninstall") ||
		strings.EqualFold(strings.TrimSpace(e.Publisher), "Electronic Arts")
}

// firstArg is the program in a command line, quoted or not.
func firstArg(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if strings.HasPrefix(cmd, `"`) {
		if i := strings.Index(cmd[1:], `"`); i >= 0 {
			return cmd[1 : i+1]
		}
	}
	if i := strings.Index(strings.ToLower(cmd), ".exe"); i >= 0 {
		return cmd[:i+4]
	}
	return cmd
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// ------------------------------------------------------------------ Xbox ---

type msGameConfig struct {
	Identity struct {
		Name      string `xml:"Name,attr"`
		Publisher string `xml:"Publisher,attr"`
	} `xml:"Identity"`
	ShellVisuals struct {
		DefaultDisplayName string `xml:"DefaultDisplayName,attr"`
	} `xml:"ShellVisuals"`
	Executables []struct {
		Name string `xml:"Name,attr"`
		ID   string `xml:"Id,attr"`
	} `xml:"ExecutableList>Executable"`
}

// XboxGames lists games the Xbox app installed in the given XboxGames
// folders: each one has Content\MicrosoftGame.config. They start through
// Windows (shell:AppsFolder), which is how the Xbox app starts them.
func XboxGames(roots []string) []FoundGame {
	var out []FoundGame
	for _, root := range roots {
		ents, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, d := range ents {
			if !d.IsDir() {
				continue
			}
			dir := filepath.Join(root, d.Name())
			content := filepath.Join(dir, "Content")
			b, err := os.ReadFile(filepath.Join(content, "MicrosoftGame.config"))
			if err != nil || len(b) > 4<<20 {
				continue
			}
			var c msGameConfig
			if xml.Unmarshal(b, &c) != nil || c.Identity.Name == "" || c.Identity.Publisher == "" || len(c.Executables) == 0 {
				continue
			}
			exe := c.Executables[0]
			name := c.ShellVisuals.DefaultDisplayName
			if name == "" || strings.HasPrefix(strings.ToLower(name), "ms-resource:") {
				name = d.Name()
			}
			f := FoundGame{Key: "dir:" + strings.ToLower(filepath.Clean(dir)), Name: name, Source: "xbox", Dir: filepath.Clean(dir),
				ProcessName: strings.TrimSuffix(filepath.Base(exe.Name), filepath.Ext(exe.Name))}
			if exe.ID != "" {
				f.URI = `shell:AppsFolder\` + c.Identity.Name + "_" + PublisherID(c.Identity.Publisher) + "!" + exe.ID
			}
			if p := filepath.Join(content, filepath.FromSlash(exe.Name)); isFile(p) {
				f.Exe = p
			}
			if f.URI == "" && f.Exe == "" {
				continue
			}
			out = append(out, f)
		}
	}
	return out
}

// PublisherID is the 13-character id Windows makes from a package's
// publisher for its family name: the first 8 bytes of the SHA-256 of the
// publisher (UTF-16LE), in Crockford's base32.
func PublisherID(publisher string) string {
	u := utf16.Encode([]rune(publisher))
	b := make([]byte, 2*len(u))
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[2*i:], c)
	}
	sum := sha256.Sum256(b)
	v := binary.BigEndian.Uint64(sum[:8])
	const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"
	out := make([]byte, 13)
	// 64 bits and one 0 bit make 13 groups of 5.
	for i := 0; i < 13; i++ {
		shift := 59 - 5*i
		var g uint64
		if shift >= 0 {
			g = (v >> uint(shift)) & 31
		} else {
			g = (v << uint(-shift)) & 31
		}
		out[i] = alphabet[g]
	}
	return string(out)
}

// ----------------------------------------------------------------- itch ---

// ItchGames lists games the itch app installed in appsDir
// (%APPDATA%\itch\apps): each has .itch\receipt.json.gz with its title.
func ItchGames(appsDir string) []FoundGame {
	ents, err := os.ReadDir(appsDir)
	if err != nil {
		return nil
	}
	var out []FoundGame
	for _, d := range ents {
		if !d.IsDir() {
			continue
		}
		dir := filepath.Join(appsDir, d.Name())
		name := itchTitle(filepath.Join(dir, ".itch", "receipt.json.gz"))
		if name == "" {
			continue
		}
		exe, process := MainExe(dir)
		if exe == "" {
			continue
		}
		out = append(out, FoundGame{Key: "dir:" + strings.ToLower(filepath.Clean(dir)), Name: name, Source: "itch", Dir: filepath.Clean(dir), Exe: exe, ProcessName: process})
	}
	return out
}

func itchTitle(receipt string) string {
	f, err := os.Open(receipt)
	if err != nil {
		return ""
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return ""
	}
	defer zr.Close()
	var r struct {
		Game struct {
			Title string `json:"title"`
		} `json:"game"`
	}
	b, err := io.ReadAll(io.LimitReader(zr, 4<<20))
	if err != nil || json.Unmarshal(b, &r) != nil {
		return ""
	}
	return strings.TrimSpace(r.Game.Title)
}

// ------------------------------------------------------------ launching ---

var launchURIs = []*regexp.Regexp{
	regexp.MustCompile(`^uplay://launch/\d{1,10}/\d{1,4}$`),
	regexp.MustCompile(`^battlenet://[A-Za-z0-9]{1,12}$`),
	regexp.MustCompile(`^amazon-games://play/[A-Za-z0-9-]{1,80}$`),
	regexp.MustCompile(`^shell:AppsFolder\\[A-Za-z0-9.-]{1,120}_[0-9a-hjkmnp-tv-z]{13}![A-Za-z0-9._-]{1,80}$`),
	regexp.MustCompile(`^com\.epicgames\.launcher://apps/[A-Za-z0-9%._-]{1,300}\?action=launch&silent=true$`),
}

// SafeLaunchURI accepts only the launch links of the launchers Lootnest
// knows; anything else (a file:, a web page, another program's link) is
// never opened as a game.
func SafeLaunchURI(u string) bool {
	for _, re := range launchURIs {
		if re.MatchString(u) {
			return true
		}
	}
	return false
}
