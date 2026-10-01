package main

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"
	"time"
)

// viewItems returns every open item plus the items closed today:
// closed first by closed_at, then open by created_at.
func (a *app) viewItems(s *store) ([]*item, error) {
	now := a.now()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return s.query(`closed_at IS NULL OR closed_at >= ?`,
		`closed_at IS NULL, closed_at, created_at, id`, midnight.UTC().Format(time.RFC3339))
}

func (a *app) renderView(w io.Writer, items []*item) {
	open := 0
	for _, it := range items {
		if it.State == "open" {
			open++
		}
	}
	if len(items) == 0 {
		fmt.Fprintln(w, "nothing open")
		return
	}
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	for i, it := range items {
		mark := "○"
		if it.State == "closed" {
			mark = "✓"
		}
		since := ""
		if n := commitsSince(it); n != nil && it.State == "open" {
			since = fmt.Sprintf("%d commits since", *n)
			if *n == 1 {
				since = "1 commit since"
			}
		}
		fmt.Fprintf(tw, "%3d  %s  %s %s\t%s\t%s\t%s\t%s\n",
			i+1, it.Hash, mark, it.Text, dash(repoName(it.Repo)), dash(it.Branch), a.when(it.CreatedAt), since)
	}
	tw.Flush()
	for line := range strings.Lines(buf.String()) {
		fmt.Fprintln(w, strings.TrimRight(line, " \n"))
	}
	fmt.Fprintf(w, "\n%d open, %d closed\n", open, len(items)-open)
}

// when shows HH:MM for today and YYYY-MM-DD for earlier days.
func (a *app) when(t time.Time) string {
	now := a.now()
	t = t.In(now.Location())
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}
	return t.Format("2006-01-02")
}

func repoName(repo string) string {
	if repo == "" {
		return ""
	}
	return filepath.Base(repo)
}

func dash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// showItem prints every stored field as "name: value", one per line.
func showItem(w io.Writer, it *item) {
	closed := ""
	if it.ClosedAt != nil {
		closed = it.ClosedAt.Format(time.RFC3339)
	}
	for _, f := range [][2]string{
		{"hash", it.Hash}, {"text", it.Text}, {"state", it.State},
		{"created_at", it.CreatedAt.Format(time.RFC3339)}, {"closed_at", closed},
		{"cwd", it.Cwd}, {"repo", it.Repo}, {"branch", it.Branch}, {"sha", it.SHA},
		{"tmux", it.Tmux}, {"tty", it.TTY},
		{"herdr_workspace", it.HerdrWorkspace}, {"herdr_tab", it.HerdrTab},
	} {
		fmt.Fprintf(w, "%s: %s\n", f[0], f[1])
	}
	for _, k := range slices.Sorted(maps.Keys(it.Env)) {
		fmt.Fprintf(w, "env.%s: %s\n", k, it.Env[k])
	}
}

type viewEntry struct {
	Position     int  `json:"position"`
	CommitsSince *int `json:"commits_since"`
	*item        `json:",inline"`
}

func (a *app) viewJSON(items []*item) map[string]any {
	entries := make([]viewEntry, len(items))
	open := 0
	for i, it := range items {
		entries[i] = viewEntry{Position: i + 1, item: it}
		if it.State == "open" {
			open++
			entries[i].CommitsSince = commitsSince(it)
		}
	}
	return map[string]any{"items": entries, "open": open, "closed": len(items) - open}
}
