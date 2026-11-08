package core

import (
	"encoding/json"
	"errors"
	"os"
)

// Records by Library key, such as what Lootnest installed from itch.io
// (itch.json) or GOG (gog.json). The callers hold s.mu.

func readRecords[T any](path string) map[string]T {
	out := map[string]T{}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if out == nil {
		out = map[string]T{}
	}
	return out
}

// readRecordsToChange reads the records before a write. A file that
// doesn't parse is kept as <name>.broken, so the write never replaces it.
func readRecordsToChange[T any](path string) (map[string]T, error) {
	out := map[string]T{}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if json.Unmarshal(b, &out) != nil {
		if err := os.Rename(path, path+".broken"); err != nil {
			return nil, err
		}
		out = nil
	}
	if out == nil {
		out = map[string]T{}
	}
	return out, nil
}

func changeRecords[T any](s *Store, path string, change func(map[string]T)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	recs, err := readRecordsToChange[T](path)
	if err != nil {
		return err
	}
	change(recs)
	return writeJSONAtomic(path, recs)
}

// rekeyRecords moves the record of from to to, unless to has one already.
func rekeyRecords[T any](path, from, to string) error {
	recs, err := readRecordsToChange[T](path)
	if err != nil {
		return err
	}
	r, ok := recs[from]
	if !ok {
		return nil
	}
	if _, has := recs[to]; !has {
		recs[to] = r
	}
	delete(recs, from)
	return writeJSONAtomic(path, recs)
}
