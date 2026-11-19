package core

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
)

// A read-only reader for one table of a SQLite 3 database, for Lutris's
// library (pga.db) on Linux. It follows the file format
// (https://www.sqlite.org/fileformat2.html): table b-trees from the root
// page sqlite_master names, records, overflow pages. It never writes, takes
// no lock and doesn't read the -wal file, so rows a running program hasn't
// checkpointed yet are missed.

// sqliteMax is the largest database file read.
const sqliteMax = 256 << 20

type sqliteDB struct {
	data   []byte
	page   int // page size
	usable int // page size less the reserved bytes at each page's end
}

func openSQLite(path string) (*sqliteDB, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Size() > sqliteMax {
		return nil, fmt.Errorf("%s is too large to read (%d MB)", path, st.Size()>>20)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseSQLite(b)
}

func parseSQLite(b []byte) (*sqliteDB, error) {
	if len(b) < 100 || string(b[:16]) != "SQLite format 3\x00" {
		return nil, errors.New("not a SQLite 3 database")
	}
	ps := int(binary.BigEndian.Uint16(b[16:18]))
	if ps == 1 {
		ps = 65536
	}
	if ps < 512 || ps&(ps-1) != 0 {
		return nil, errors.New("SQLite database with a bad page size")
	}
	if enc := binary.BigEndian.Uint32(b[56:60]); enc != 0 && enc != 1 {
		return nil, errors.New("SQLite database in UTF-16, which isn't read")
	}
	usable := ps - int(b[20])
	if usable < 480 {
		return nil, errors.New("SQLite database with too little room per page")
	}
	return &sqliteDB{data: b, page: ps, usable: usable}, nil
}

func (db *sqliteDB) pageAt(n uint32) ([]byte, error) {
	start := (int(n) - 1) * db.page
	if n == 0 || start < 0 || start+db.page > len(db.data) {
		return nil, fmt.Errorf("SQLite page %d is outside the file", n)
	}
	return db.data[start : start+db.page], nil
}

// sqliteRow is one row: its rowid and its values (int64, float64, string,
// []byte or nil) in column order.
type sqliteRow struct {
	rowid  int64
	values []any
}

// rows walks a table b-tree from its root page, in rowid order.
func (db *sqliteDB) rows(root uint32) ([]sqliteRow, error) {
	var out []sqliteRow
	seen := map[uint32]bool{}
	var walk func(n uint32, depth int) error
	walk = func(n uint32, depth int) error {
		if seen[n] || depth > 40 {
			return errors.New("SQLite table pages loop")
		}
		seen[n] = true
		pg, err := db.pageAt(n)
		if err != nil {
			return err
		}
		h := 0
		if n == 1 {
			h = 100 // the file header
		}
		kind := pg[h]
		cells := int(binary.BigEndian.Uint16(pg[h+3 : h+5]))
		switch kind {
		case 0x05: // interior: left children, then the right-most
			ptrs := h + 12
			for i := 0; i < cells; i++ {
				off, err := cellOffset(pg, ptrs, i)
				if err != nil || off+4 > len(pg) {
					return errors.New("SQLite page with a bad cell")
				}
				if err := walk(binary.BigEndian.Uint32(pg[off:off+4]), depth+1); err != nil {
					return err
				}
			}
			return walk(binary.BigEndian.Uint32(pg[h+8:h+12]), depth+1)
		case 0x0D: // leaf
			ptrs := h + 8
			for i := 0; i < cells; i++ {
				off, err := cellOffset(pg, ptrs, i)
				if err != nil {
					return err
				}
				row, err := db.leafCell(pg, off)
				if err != nil {
					return err
				}
				out = append(out, row)
			}
			return nil
		}
		return fmt.Errorf("SQLite page %d isn't a table page", n)
	}
	return out, walk(root, 0)
}

func cellOffset(pg []byte, ptrs, i int) (int, error) {
	p := ptrs + 2*i
	if p+2 > len(pg) {
		return 0, errors.New("SQLite page with a bad cell")
	}
	off := int(binary.BigEndian.Uint16(pg[p : p+2]))
	if off >= len(pg) {
		return 0, errors.New("SQLite page with a bad cell")
	}
	return off, nil
}

// leafCell reads a table leaf cell: payload size, rowid, payload (spilling
// into overflow pages when it's large).
func (db *sqliteDB) leafCell(pg []byte, off int) (sqliteRow, error) {
	size, n := sqliteVarint(pg[off:])
	if n == 0 {
		return sqliteRow{}, errors.New("SQLite cell with a bad size")
	}
	off += n
	rowid, n := sqliteVarint(pg[off:])
	if n == 0 {
		return sqliteRow{}, errors.New("SQLite cell with a bad rowid")
	}
	off += n
	total := int(size)
	if size > uint64(len(db.data)) {
		return sqliteRow{}, errors.New("SQLite cell larger than the file")
	}
	u := db.usable
	maxLocal := u - 35
	local := total
	if total > maxLocal {
		minLocal := (u-12)*32/255 - 23
		local = minLocal + (total-minLocal)%(u-4)
		if local > maxLocal {
			local = minLocal
		}
	}
	if off+local > len(pg) {
		return sqliteRow{}, errors.New("SQLite cell past its page")
	}
	payload := append([]byte(nil), pg[off:off+local]...)
	if local < total {
		if off+local+4 > len(pg) {
			return sqliteRow{}, errors.New("SQLite cell past its page")
		}
		next := binary.BigEndian.Uint32(pg[off+local : off+local+4])
		seen := map[uint32]bool{}
		for len(payload) < total {
			if next == 0 || seen[next] {
				return sqliteRow{}, errors.New("SQLite overflow pages end early")
			}
			seen[next] = true
			op, err := db.pageAt(next)
			if err != nil {
				return sqliteRow{}, err
			}
			take := min(total-len(payload), u-4)
			payload = append(payload, op[4:4+take]...)
			next = binary.BigEndian.Uint32(op[:4])
		}
	}
	vals, err := sqliteRecord(payload)
	return sqliteRow{rowid: int64(rowid), values: vals}, err
}

// sqliteVarint reads SQLite's big-endian varint (1 to 9 bytes); n is 0
// when b ends first.
func sqliteVarint(b []byte) (v uint64, n int) {
	for i := 0; i < 9; i++ {
		if i >= len(b) {
			return 0, 0
		}
		if i == 8 {
			return v<<8 | uint64(b[i]), 9
		}
		v = v<<7 | uint64(b[i]&0x7f)
		if b[i] < 0x80 {
			return v, i + 1
		}
	}
	return v, 9
}

// sqliteRecord reads a record's values.
func sqliteRecord(p []byte) ([]any, error) {
	hsize, n := sqliteVarint(p)
	if n == 0 || hsize > uint64(len(p)) {
		return nil, errors.New("SQLite record with a bad header")
	}
	var types []uint64
	for i := n; i < int(hsize); {
		t, m := sqliteVarint(p[i:int(hsize)])
		if m == 0 {
			return nil, errors.New("SQLite record with a bad header")
		}
		types = append(types, t)
		i += m
	}
	body := p[hsize:]
	out := make([]any, 0, len(types))
	for _, t := range types {
		size := 0
		switch {
		case t <= 4:
			size = int(t)
		case t == 5:
			size = 6
		case t == 6 || t == 7:
			size = 8
		case t >= 12:
			size = int((t - 12) / 2)
		}
		if size > len(body) {
			return nil, errors.New("SQLite record shorter than its header says")
		}
		v := body[:size]
		body = body[size:]
		switch {
		case t == 0:
			out = append(out, nil)
		case t <= 6:
			var x int64
			for _, c := range v {
				x = x<<8 | int64(c)
			}
			if shift := 64 - 8*size; size > 0 {
				x = x << shift >> shift // sign-extend
			}
			out = append(out, x)
		case t == 7:
			out = append(out, math.Float64frombits(binary.BigEndian.Uint64(v)))
		case t == 8:
			out = append(out, int64(0))
		case t == 9:
			out = append(out, int64(1))
		case t >= 12 && t%2 == 0:
			out = append(out, append([]byte(nil), v...))
		case t >= 13:
			out = append(out, string(v))
		default:
			return nil, errors.New("SQLite record with a reserved type")
		}
	}
	return out, nil
}

// sqliteTable reads every row of a table as column name -> value. A column
// added after a row was written is missing from that row (nil); the INTEGER
// PRIMARY KEY column is the rowid.
func sqliteTable(path, table string) ([]map[string]any, error) {
	db, err := openSQLite(path)
	if err != nil {
		return nil, err
	}
	master, err := db.rows(1)
	if err != nil {
		return nil, err
	}
	var root uint32
	var create string
	for _, r := range master {
		if len(r.values) < 5 {
			continue
		}
		typ, _ := r.values[0].(string)
		name, _ := r.values[1].(string)
		page, _ := r.values[3].(int64)
		if typ == "table" && strings.EqualFold(name, table) && page > 0 && page <= math.MaxUint32 {
			root = uint32(page)
			create, _ = r.values[4].(string)
		}
	}
	if root == 0 {
		return nil, fmt.Errorf("no table %s in the database", table)
	}
	cols, key := sqliteColumns(create)
	rows, err := db.rows(root)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		m := make(map[string]any, len(cols))
		for i, c := range cols {
			if i < len(r.values) {
				m[c] = r.values[i]
			}
			if i == key {
				m[c] = r.rowid
			}
		}
		out = append(out, m)
	}
	return out, nil
}

// sqliteColumns reads the column names from a CREATE TABLE statement, and
// which one (if any) is the INTEGER PRIMARY KEY that stands for the rowid.
func sqliteColumns(create string) (cols []string, key int) {
	key = -1
	open, end := strings.Index(create, "("), strings.LastIndex(create, ")")
	if open < 0 || end <= open {
		return nil, key
	}
	var parts []string
	depth, start := 0, open+1
	var quote byte
	for i := open + 1; i < end; i++ {
		c := create[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"', '`':
			quote = c
		case '[':
			quote = ']'
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, create[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, create[start:end])
	for _, p := range parts {
		f := strings.Fields(p)
		if len(f) == 0 {
			continue
		}
		switch strings.ToUpper(f[0]) {
		case "PRIMARY", "UNIQUE", "CHECK", "FOREIGN", "CONSTRAINT":
			continue // a table constraint, not a column
		}
		name := strings.Trim(f[0], "\"`[]'")
		up := strings.ToUpper(strings.Join(f[1:], " "))
		if strings.HasPrefix(up, "INTEGER") && strings.Contains(up, "PRIMARY KEY") && !strings.Contains(up, "DESC") {
			key = len(cols)
		}
		cols = append(cols, name)
	}
	return cols, key
}
