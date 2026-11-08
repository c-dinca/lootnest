package core

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// jsonObj is a JSON object with its keys lowercased and without
// underscores, so snake_case and camelCase read the same: the forgiving
// reader for the stores' replies (itch.io, GOG).
type jsonObj map[string]json.RawMessage

func normKey(k string) string { return strings.ToLower(strings.ReplaceAll(k, "_", "")) }

var errNotObject = errors.New("not a JSON object")

func decodeObj(b []byte) (jsonObj, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, errNotObject
	}
	o := jsonObj{}
	for k, v := range raw {
		o[normKey(k)] = v
	}
	return o, nil
}

func (o jsonObj) obj(k string) jsonObj {
	x, err := decodeObj(o[k])
	if err != nil {
		return jsonObj{}
	}
	return x
}

func (o jsonObj) list(k string) []jsonObj {
	var raw []json.RawMessage
	_ = json.Unmarshal(o[k], &raw)
	var out []jsonObj
	for _, r := range raw {
		if x, err := decodeObj(r); err == nil {
			out = append(out, x)
		}
	}
	return out
}

func (o jsonObj) strs(k string) []string {
	var out []string
	if json.Unmarshal(o[k], &out) != nil {
		return nil
	}
	return out
}

func (o jsonObj) str(k string) string {
	var s string
	if json.Unmarshal(o[k], &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(o[k], &n) == nil {
		return n.String()
	}
	return ""
}

func (o jsonObj) num(k string) int64 {
	var n json.Number
	if json.Unmarshal(o[k], &n) != nil {
		var s string
		if json.Unmarshal(o[k], &s) != nil {
			return 0
		}
		n = json.Number(strings.TrimSpace(s))
	}
	if i, err := n.Int64(); err == nil {
		return i
	}
	f, _ := n.Float64()
	return int64(f)
}

func (o jsonObj) flag(k string) bool {
	var b bool
	if json.Unmarshal(o[k], &b) == nil {
		return b
	}
	switch strings.ToLower(o.str(k)) {
	case "true", "1", "yes":
		return true
	}
	return false
}

func (o jsonObj) when(k string) time.Time {
	s := o.str(k)
	for _, f := range []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02 15:04:05.999999"} {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
