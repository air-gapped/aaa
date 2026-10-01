package main

import (
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type item struct {
	Hash           string            `json:"hash"`
	Text           string            `json:"text"`
	State          string            `json:"state"`
	CreatedAt      time.Time         `json:"created_at"`
	ClosedAt       *time.Time        `json:"closed_at"`
	Cwd            string            `json:"cwd"`
	Repo           string            `json:"repo"`
	Branch         string            `json:"branch"`
	SHA            string            `json:"sha"`
	Tmux           string            `json:"tmux"`
	TTY            string            `json:"tty"`
	HerdrWorkspace string            `json:"herdr_workspace"`
	HerdrTab       string            `json:"herdr_tab"`
	Env            map[string]string `json:"env"`
}

const schema = `
CREATE TABLE items (
	id              INTEGER PRIMARY KEY,
	hash            TEXT NOT NULL UNIQUE,
	text            TEXT NOT NULL,
	created_at      TEXT NOT NULL,
	closed_at       TEXT,
	cwd             TEXT NOT NULL DEFAULT '',
	repo            TEXT NOT NULL DEFAULT '',
	branch          TEXT NOT NULL DEFAULT '',
	sha             TEXT NOT NULL DEFAULT '',
	tmux            TEXT NOT NULL DEFAULT '',
	tty             TEXT NOT NULL DEFAULT '',
	herdr_workspace TEXT NOT NULL DEFAULT '',
	herdr_tab       TEXT NOT NULL DEFAULT '',
	env             TEXT NOT NULL DEFAULT '{}'
);`

type store struct{ db *sql.DB }

func openStore(env map[string]string) (*store, error) {
	dir := env["XDG_DATA_HOME"]
	if dir == "" {
		dir = filepath.Join(env["HOME"], ".local", "share")
	}
	dir = filepath.Join(dir, "aaa")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "aaa.db")+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return nil, err
	}
	if version == 0 {
		if _, err := db.Exec(schema + `PRAGMA user_version = 1;`); err != nil {
			return nil, err
		}
	}
	return &store{db}, nil
}

func (s *store) insert(it *item) error {
	env, err := json.Marshal(it.Env)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO items
		(hash, text, created_at, cwd, repo, branch, sha, tmux, tty, herdr_workspace, herdr_tab, env)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		it.Hash, it.Text, it.CreatedAt.UTC().Format(time.RFC3339), it.Cwd, it.Repo, it.Branch, it.SHA,
		it.Tmux, it.TTY, it.HerdrWorkspace, it.HerdrTab, string(env))
	return err
}

const itemColumns = `hash, text, created_at, closed_at, cwd, repo, branch, sha, tmux, tty, herdr_workspace, herdr_tab, env`

// query returns items matching where, in the given order.
func (s *store) query(where, order string, args ...any) ([]*item, error) {
	rows, err := s.db.Query(`SELECT `+itemColumns+` FROM items WHERE `+where+` ORDER BY `+order, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*item
	for rows.Next() {
		var it item
		var created, env string
		var closed sql.NullString
		if err := rows.Scan(&it.Hash, &it.Text, &created, &closed, &it.Cwd, &it.Repo, &it.Branch, &it.SHA,
			&it.Tmux, &it.TTY, &it.HerdrWorkspace, &it.HerdrTab, &env); err != nil {
			return nil, err
		}
		it.CreatedAt, _ = time.Parse(time.RFC3339, created)
		it.State = "open"
		if closed.Valid {
			t, _ := time.Parse(time.RFC3339, closed.String)
			it.ClosedAt, it.State = &t, "closed"
		}
		if err := json.Unmarshal([]byte(env), &it.Env); err != nil {
			return nil, err
		}
		items = append(items, &it)
	}
	return items, rows.Err()
}

func (s *store) hashTaken(h string) bool {
	var n int
	s.db.QueryRow(`SELECT count(*) FROM items WHERE hash = ?`, h).Scan(&n)
	return n > 0
}

var errNoItem = errors.New("no item")

// get returns the item with hash h, or an error naming h.
func (s *store) get(h string) (*item, error) {
	h = strings.ToLower(h)
	items, err := s.query(`hash = ?`, `id`, h)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%w %s", errNoItem, h)
	}
	return items[0], nil
}

// close closes item h and reports whether it was open.
func (s *store) close(h string, now time.Time) (*item, bool, error) {
	res, err := s.db.Exec(`UPDATE items SET closed_at = ? WHERE hash = ? AND closed_at IS NULL`,
		now.UTC().Format(time.RFC3339), strings.ToLower(h))
	if err != nil {
		return nil, false, err
	}
	n, _ := res.RowsAffected()
	it, err := s.get(h)
	return it, n > 0, err
}

// closeAll closes every open item and returns them.
func (s *store) closeAll(now time.Time) ([]*item, error) {
	rows, err := s.db.Query(`UPDATE items SET closed_at = ? WHERE closed_at IS NULL RETURNING hash`,
		now.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	var hashes []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			rows.Close()
			return nil, err
		}
		hashes = append(hashes, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var items []*item
	for _, h := range hashes {
		it, err := s.get(h)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

// edit replaces the text of item h.
func (s *store) edit(h, text string) (*item, error) {
	if _, err := s.db.Exec(`UPDATE items SET text = ? WHERE hash = ?`, text, strings.ToLower(h)); err != nil {
		return nil, err
	}
	return s.get(h)
}
