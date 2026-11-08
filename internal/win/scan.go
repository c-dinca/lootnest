//go:build windows

package win

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// FixedDrives returns local hard-drive roots like C:\ and D:\.
func FixedDrives() []string {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}
	var out []string
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		if windows.GetDriveType(windows.StringToUTF16Ptr(root)) == windows.DRIVE_FIXED {
			out = append(out, root)
		}
	}
	return out
}

type GOGGame struct{ ID, Name, Path, Exe string }

// GOGGames lists games installed through GOG (from the registry).
func GOGGames() []GOGGame {
	var out []GOGGame
	for _, sub := range []string{`SOFTWARE\WOW6432Node\GOG.com\Games`, `SOFTWARE\GOG.com\Games`} {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, sub, registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			continue
		}
		names, _ := k.ReadSubKeyNames(-1)
		k.Close()
		for _, n := range names {
			g, err := registry.OpenKey(registry.LOCAL_MACHINE, sub+`\`+n, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			name, _, _ := g.GetStringValue("gameName")
			path, _, _ := g.GetStringValue("path")
			exe, _, _ := g.GetStringValue("exe")
			g.Close()
			if name != "" && path != "" && exe != "" {
				out = append(out, GOGGame{n, name, path, exe})
			}
		}
		if len(out) > 0 {
			break
		}
	}
	return out
}

// ShortcutFiles lists .lnk files on the desktops and in the Start menus.
func ShortcutFiles() []string {
	var dirs []string
	for _, id := range []*windows.KNOWNFOLDERID{windows.FOLDERID_Desktop, windows.FOLDERID_PublicDesktop, windows.FOLDERID_Programs, windows.FOLDERID_CommonPrograms} {
		if p := KnownFolder(id); p != "" {
			dirs = append(dirs, p)
		}
	}
	var out []string
	for _, d := range dirs {
		base := strings.Count(d, `\`)
		_ = filepath.WalkDir(d, func(p string, e os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if e.IsDir() && strings.Count(p, `\`)-base > 2 {
				return filepath.SkipDir
			}
			if !e.IsDir() && strings.EqualFold(filepath.Ext(p), ".lnk") {
				out = append(out, p)
			}
			return nil
		})
	}
	return out
}

// ResolveShortcuts returns each .lnk's target path (only those that resolve).
func ResolveShortcuts(lnks []string) map[string]string {
	out := map[string]string{}
	_ = withCOM(func() error {
		for _, l := range lnks {
			link, err := createInstance(clsidShellLink, iidShellLinkW)
			if err != nil {
				return err
			}
			var pf *comObj
			if hr(link.call(0, uintptr(unsafe.Pointer(iidPersistFile)), uintptr(unsafe.Pointer(&pf)))) == nil {
				// IPersistFile::Load(path, STGM_READ)
				if hr(pf.call(5, uintptr(unsafe.Pointer(wstr(l))), 0)) == nil {
					buf := make([]uint16, 1024)
					// IShellLinkW::GetPath(buf, len, NULL, SLGP_RAWPATH=4)
					if hr(link.call(3, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0, 4)) == nil {
						t := windows.UTF16ToString(buf)
						if t != "" {
							if exp, err := registry.ExpandString(t); err == nil {
								t = exp
							}
							out[l] = t
						}
					}
				}
				pf.release()
			}
			link.release()
		}
		return nil
	})
	return out
}

func ProgramData() string { return KnownFolder(windows.FOLDERID_ProgramData) }

// InstalledProgram is one program from Windows' installed programs list.
type InstalledProgram struct {
	Key, DisplayName, Publisher, InstallLocation, UninstallString, DisplayIcon string
}

// InstalledPrograms lists the installed programs (for every user and for
// this one, 64- and 32-bit).
func InstalledPrograms() []InstalledProgram {
	var out []InstalledProgram
	for _, src := range []struct {
		root registry.Key
		path string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`},
		{registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Uninstall`},
	} {
		k, err := registry.OpenKey(src.root, src.path, registry.ENUMERATE_SUB_KEYS)
		if err != nil {
			continue
		}
		names, _ := k.ReadSubKeyNames(-1)
		k.Close()
		for _, n := range names {
			sk, err := registry.OpenKey(src.root, src.path+`\`+n, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			get := func(v string) string { s, _, _ := sk.GetStringValue(v); return s }
			out = append(out, InstalledProgram{Key: n, DisplayName: get("DisplayName"), Publisher: get("Publisher"),
				InstallLocation: get("InstallLocation"), UninstallString: get("UninstallString"), DisplayIcon: get("DisplayIcon")})
			sk.Close()
		}
	}
	return out
}

// XboxGameFolders are the folders the Xbox app installs games in: the one
// each drive's .GamingRoot file names, and XboxGames on every drive.
func XboxGameFolders() []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if st, err := os.Stat(p); err == nil && st.IsDir() && !seen[strings.ToLower(p)] {
			seen[strings.ToLower(p)] = true
			out = append(out, p)
		}
	}
	for _, d := range FixedDrives() {
		// .GamingRoot: "RGBX", a count, then UTF-16 folder names.
		if b, err := os.ReadFile(filepath.Join(d, ".GamingRoot")); err == nil && len(b) > 8 && string(b[:4]) == "RGBX" {
			u := make([]uint16, 0, (len(b)-8)/2)
			for i := 8; i+1 < len(b); i += 2 {
				u = append(u, uint16(b[i])|uint16(b[i+1])<<8)
			}
			for _, name := range strings.Split(windows.UTF16ToString(u), "\x00") {
				if name = strings.TrimSpace(name); name != "" && !strings.ContainsAny(name, `:*?"<>|`) {
					add(filepath.Join(d, name))
				}
			}
		}
		add(filepath.Join(d, "XboxGames"))
	}
	return out
}

// ReadRegKey reads one key under HKCU or HKLM, read-only (KEY_READ): its
// subkeys' names and its text and number values as text (core.RegRead).
func ReadRegKey(key string) ([]string, map[string]string, error) {
	root, path, _ := strings.Cut(key, `\`)
	var r registry.Key
	switch strings.ToUpper(root) {
	case "HKCU":
		r = registry.CURRENT_USER
	case "HKLM":
		r = registry.LOCAL_MACHINE
	default:
		return nil, nil, registry.ErrNotExist
	}
	k, err := registry.OpenKey(r, path, registry.READ)
	if err != nil {
		return nil, nil, err
	}
	defer k.Close()
	subs, _ := k.ReadSubKeyNames(2000)
	names, _ := k.ReadValueNames(200)
	vals := map[string]string{}
	for _, n := range names {
		if s, typ, err := k.GetStringValue(n); err == nil {
			if typ == registry.EXPAND_SZ {
				if x, err := registry.ExpandString(s); err == nil {
					s = x
				}
			}
			vals[n] = s
		} else if v, _, err := k.GetIntegerValue(n); err == nil {
			vals[n] = strconv.FormatUint(v, 10)
		}
	}
	return subs, vals, nil
}
