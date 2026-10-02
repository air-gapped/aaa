package main

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
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

// remote holds the live PUSH/PULL state of one item's branch.
type remote struct {
	push *int
	pull *bool
}

// remotes checks every distinct repo+branch of the open items in parallel.
func remotes(items []*item) map[*item]remote {
	type key struct{ repo, branch string }
	index := map[key]int{}
	var keys []key
	for _, it := range items {
		k := key{it.Repo, it.Branch}
		if it.State != "open" || it.Repo == "" || it.Branch == "" || it.Branch == "HEAD" {
			continue
		}
		if _, seen := index[k]; !seen {
			index[k] = len(keys)
			keys = append(keys, k)
		}
	}
	states := make([]remote, len(keys))
	var wg sync.WaitGroup
	for i, k := range keys {
		wg.Go(func() { states[i].push, states[i].pull = remoteState(k.repo, k.branch) })
	}
	wg.Wait()
	out := map[*item]remote{}
	for _, it := range items {
		if i, ok := index[key{it.Repo, it.Branch}]; ok && it.State == "open" {
			out[it] = states[i]
		}
	}
	return out
}

const (
	dim, yellow, cyan, green, red, reset = "\033[2m", "\033[33m", "\033[36m", "\033[32m", "\033[31m", "\033[0m"
)

func (a *app) renderView(w io.Writer, items []*item) {
	if len(items) == 0 {
		fmt.Fprintln(w, "nothing open")
		return
	}
	c := func(code, s string) string {
		if !a.color || s == "" {
			return s
		}
		return code + s + reset
	}
	rs := remotes(items)
	tw, gw := len("TEXT"), len("GIT")
	for _, it := range items {
		tw, gw = max(tw, len([]rune(it.Text))), max(gw, len(it.Branch))
	}
	pad := func(s string, n int) string { return s + strings.Repeat(" ", max(0, n-len([]rune(s)))) }

	fmt.Fprintln(w, c(dim, strings.TrimRight(fmt.Sprintf("  %-3s  %-16s  %s  %s  PUSH  PULL  PATH",
		"ID", "STARTED", pad("TEXT", tw), pad("GIT", gw)), " ")))
	open, prevDay := 0, ""
	for _, it := range items {
		t := it.CreatedAt.In(a.now().Location())
		day := t.Format("Mon Jan 2")
		shown := day
		if day == prevDay {
			shown = ""
		}
		prevDay = day
		push, pull := "", ""
		if r := rs[it]; r.push != nil && *r.push > 0 {
			push = strconv.Itoa(*r.push)
		}
		if r := rs[it]; r.pull != nil && *r.pull {
			pull = "yes"
		}
		path := tilde(it.Cwd, a.env["HOME"])
		if it.State == "closed" {
			fmt.Fprintln(w, c(dim, strings.TrimRight(fmt.Sprintf("  %-3s  %-10s %5s  %s  %s  %4s  %4s  %s",
				it.Hash, shown, t.Format("15:04"), pad(it.Text, tw), pad(it.Branch, gw), "", "", path), " ")))
			continue
		}
		open++
		fmt.Fprintln(w, strings.TrimRight(fmt.Sprintf("  %s  %-10s %s  %s  %s  %s  %s  %s",
			c(yellow, fmt.Sprintf("%-3s", it.Hash)), shown, c(dim, t.Format("15:04")), pad(it.Text, tw),
			c(cyan, pad(it.Branch, gw)), c(green, fmt.Sprintf("%4s", push)), c(red, fmt.Sprintf("%4s", pull)),
			c(dim, path)), " "))
	}
	fmt.Fprintf(w, "\n%s\n", c(dim, fmt.Sprintf("  %d open, %d done today", open, len(items)-open)))
}

// tilde shortens home to ~ in path.
func tilde(path, home string) string {
	if home != "" && (path == home || strings.HasPrefix(path, home+"/")) {
		return "~" + path[len(home):]
	}
	return path
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
	CommitsSince *int  `json:"commits_since"`
	Push         *int  `json:"push"`
	Pull         *bool `json:"pull"`
	*item        `json:",inline"`
}

func (a *app) viewJSON(items []*item) map[string]any {
	entries := make([]viewEntry, len(items))
	rs := remotes(items)
	open := 0
	for i, it := range items {
		entries[i] = viewEntry{item: it}
		if it.State == "open" {
			open++
			entries[i].CommitsSince = commitsSince(it)
			entries[i].Push, entries[i].Pull = rs[it].push, rs[it].pull
		}
	}
	return map[string]any{"items": entries, "open": open, "closed": len(items) - open}
}
