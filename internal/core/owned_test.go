package core

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// appInfoFile writes an appinfo.vdf of the given version with apps id ->
// {name, type}, the way Steam lays it out.
func appInfoFile(t *testing.T, path string, version uint32, apps [][3]string) {
	t.Helper()
	le := binary.LittleEndian
	var strs []string
	index := map[string]uint32{}
	var out bytes.Buffer
	key := func(b *bytes.Buffer, k string) {
		if version < 0x07564429 {
			b.WriteString(k + "\x00")
			return
		}
		i, ok := index[k]
		if !ok {
			i = uint32(len(strs))
			index[k] = i
			strs = append(strs, k)
		}
		_ = binary.Write(b, le, i)
	}
	_ = binary.Write(&out, le, version)
	_ = binary.Write(&out, le, uint32(1))
	if version == 0x07564429 {
		_ = binary.Write(&out, le, int64(0)) // the table's offset, set below
	}
	fixed := 60
	if version == 0x07564427 {
		fixed = 40
	}
	for _, a := range apps {
		var d bytes.Buffer
		d.WriteByte(0)
		key(&d, "appinfo")
		d.WriteByte(2)
		key(&d, "appid")
		_ = binary.Write(&d, le, uint32(1))
		d.WriteByte(7)
		key(&d, "changenumber")
		_ = binary.Write(&d, le, uint64(9))
		d.WriteByte(0)
		key(&d, "common")
		d.WriteByte(1)
		key(&d, "name")
		d.WriteString(a[1] + "\x00")
		d.WriteByte(1)
		key(&d, "type")
		d.WriteString(a[2] + "\x00")
		d.WriteByte(8)
		d.WriteByte(8)
		d.WriteByte(8)
		var id uint32
		for _, c := range a[0] {
			id = id*10 + uint32(c-'0')
		}
		_ = binary.Write(&out, le, id)
		_ = binary.Write(&out, le, uint32(fixed+d.Len()))
		out.Write(make([]byte, fixed))
		out.Write(d.Bytes())
	}
	_ = binary.Write(&out, le, uint32(0))
	b := out.Bytes()
	if version == 0x07564429 {
		le.PutUint64(b[8:], uint64(len(b)))
		var tbl bytes.Buffer
		_ = binary.Write(&tbl, le, uint32(len(strs)))
		for _, s := range strs {
			tbl.WriteString(s + "\x00")
		}
		b = append(b, tbl.Bytes()...)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSteamOwned(t *testing.T) {
	for _, v := range []uint32{0x07564427, 0x07564428, 0x07564429} {
		steam := t.TempDir()
		user := filepath.Join(steam, "userdata", "123")
		appInfoFile(t, filepath.Join(steam, "appcache", "appinfo.vdf"), v, [][3]string{
			{"10", "Counter-Strike", "Game"},
			{"620", "Portal 2", "game"},
			{"228980", "Steamworks Common Redistributables", "Tool"},
			{"1245620", "Elden Ring", "Game"},
			{"1245621", "Elden Ring Soundtrack", "Music"},
			{"999", "Not Yours", "Game"}, // in appinfo, but not the account's
			{"413150", "Stardew Valley", "Game"},
		})
		write(t, filepath.Join(user, "config", "localconfig.vdf"), `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" {
			"10" { "LastPlayed" "1700000000" }
			"620" { "Playtime" "30" }
			"228980" { }
			"1245621" { }
			"413150" { "Playtime" "600" }
		} } } } }`)
		write(t, filepath.Join(user, "config", "librarycache", "1245620.json"), `[]`)
		write(t, filepath.Join(user, "config", "librarycache", "notes.json"), `[]`)
		got := SteamOwned(steam, user, map[int]bool{413150: true})
		want := []OwnedGame{
			{Key: "owned:steam:10", Name: "Counter-Strike", Source: "steam", AppID: 10},
			{Key: "owned:steam:1245620", Name: "Elden Ring", Source: "steam", AppID: 1245620},
			{Key: "owned:steam:620", Name: "Portal 2", Source: "steam", AppID: 620},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("version %x:\n got %+v\nwant %+v", v, got, want)
		}
	}
}

func TestSteamOwnedWithoutFiles(t *testing.T) {
	steam := t.TempDir()
	if got := SteamOwned(steam, filepath.Join(steam, "userdata", "1"), nil); got != nil {
		t.Errorf("no files: %v", got)
	}
	if got := SteamOwned("", "", nil); got != nil {
		t.Errorf("no Steam: %v", got)
	}
	// A damaged appinfo.vdf names nothing, and doesn't hang or crash.
	user := filepath.Join(steam, "userdata", "1")
	write(t, filepath.Join(user, "config", "librarycache", "620.json"), `[]`)
	write(t, filepath.Join(steam, "appcache", "appinfo.vdf"), "\x28\x44\x56\x07\x01\x00\x00\x00\x6c\x02\x00\x00\xff\xff\xff\x00")
	if got := SteamOwned(steam, user, nil); got != nil {
		t.Errorf("damaged: %v", got)
	}
}

func TestEpicOwned(t *testing.T) {
	cat := `[
	 {"id":"item1","title":"Hades","namespace":"min","categories":[{"path":"games"},{"path":"applications"}],"releaseInfo":[{"appId":"Min"}]},
	 {"id":"item2","title":"Hades: Soundtrack","namespace":"min","categories":[{"path":"addons"}],"releaseInfo":[{"appId":"MinOST"}],"mainGameItem":{"id":"item1"}},
	 {"id":"item3","title":"A DLC filed as a game","namespace":"min","categories":[{"path":"games"}],"releaseInfo":[{"appId":"MinDLC"}],"mainGameItem":{"id":"item1"}},
	 {"id":"item4","title":"Control","namespace":"calluna","categories":[{"path":"games"}],"releaseInfo":[{"appId":"Calluna"}],"mainGameItem":null},
	 {"id":"item5","title":"Installed One","namespace":"inst","categories":[{"path":"games"}],"releaseInfo":[{"appId":"Inst"}]},
	 {"id":"item6","title":"Unreal asset","namespace":"ue","categories":[{"path":"assets"}],"releaseInfo":[{"appId":"Asset"}]},
	 {"id":"item7","title":"No release","namespace":"x","categories":[{"path":"games"}]},
	 {"id":"item8","title":"Odd id","namespace":"x y","categories":[{"path":"games"}],"releaseInfo":[{"appId":"Odd?"}]}
	]`
	p := filepath.Join(t.TempDir(), "catcache.bin")
	write(t, p, base64.StdEncoding.EncodeToString([]byte(cat)))
	got := EpicOwned(p, map[string]bool{EpicID("inst", "item5", "Inst"): true})
	want := []OwnedGame{
		{Key: "owned:epic:Calluna", Name: "Control", Source: "epic", StoreID: "calluna%3Aitem4%3ACalluna"},
		{Key: "owned:epic:Min", Name: "Hades", Source: "epic", StoreID: "min%3Aitem1%3AMin"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("\n got %+v\nwant %+v", got, want)
	}
	if u := StoreAction("epic", got[1].StoreID, "install"); u != "com.epicgames.launcher://apps/min%3Aitem1%3AMin?action=install" {
		t.Errorf("install link %q", u)
	}
	write(t, p, "not base64 at all")
	if got := EpicOwned(p, nil); got != nil {
		t.Errorf("damaged: %v", got)
	}
	if got := EpicOwned(filepath.Join(t.TempDir(), "none.bin"), nil); got != nil {
		t.Errorf("missing: %v", got)
	}
}
