package main

import (
	"bytes"
	"encoding/json/v2"
	"maps"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testApp runs aaa in-process against a private data directory.
type testApp struct {
	t   *testing.T
	app *app
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	return &testApp{t: t, app: &app{
		env:  map[string]string{"XDG_DATA_HOME": t.TempDir()},
		cwd:  t.TempDir(),
		now:  func() time.Time { return time.Date(2026, 10, 1, 9, 14, 0, 0, time.Local) },
		rand: rand.New(rand.NewPCG(1, 2)),
	}}
}

// run executes aaa with args and returns stdout, stderr and the exit code.
func (ta *testApp) run(args ...string) (string, string, int) {
	ta.t.Helper()
	var out, errOut bytes.Buffer
	ta.app.stdout, ta.app.stderr = &out, &errOut
	code := ta.app.run(args)
	return out.String(), errOut.String(), code
}

func TestEmptyViewSaysNothingOpen(t *testing.T) {
	out, _, code := newTestApp(t).run()
	if code != 0 || strings.TrimSpace(out) != "nothing open" {
		t.Fatalf("got code %d, out %q; want 0, %q", code, out, "nothing open")
	}
}

func TestCapturedItemAppearsInView(t *testing.T) {
	ta := newTestApp(t)
	out, _, code := ta.run("update", "the", "readme", "file")
	if code != 0 || !strings.HasSuffix(strings.TrimSpace(out), "update the readme file") {
		t.Fatalf("capture: code %d, out %q", code, out)
	}
	hash := strings.Fields(out)[0]

	view, _, _ := ta.run()
	if !strings.Contains(view, hash+"  ○ update the readme file") {
		t.Errorf("view missing item %s:\n%s", hash, view)
	}
	if !strings.Contains(view, "1 open, 0 closed") {
		t.Errorf("view missing footer:\n%s", view)
	}
}

// capture adds an item and returns its hash.
func (ta *testApp) capture(words ...string) string {
	ta.t.Helper()
	out, errOut, code := ta.run(words...)
	if code != 0 {
		ta.t.Fatalf("capture %q: code %d, stderr %q", words, code, errOut)
	}
	return strings.Fields(out)[0]
}

func TestCloseMarksItemDoneInView(t *testing.T) {
	ta := newTestApp(t)
	h := ta.capture("update the README")

	out, _, code := ta.run("--close", h)
	if code != 0 || strings.TrimSpace(out) != "closed "+h+"  update the README" {
		t.Fatalf("close: code %d, out %q", code, out)
	}
	view, _, _ := ta.run()
	if !strings.Contains(view, h+"  ✓ update the README") || !strings.Contains(view, "0 open, 1 closed") {
		t.Errorf("view after close:\n%s", view)
	}
}

func TestCloseUnknownHashFails(t *testing.T) {
	_, errOut, code := newTestApp(t).run("--close", "zzz")
	if code != 1 || !strings.Contains(errOut, "no item zzz") {
		t.Fatalf("code %d, stderr %q; want 1, no item zzz", code, errOut)
	}
}

func TestCloseTwiceSaysAlreadyClosed(t *testing.T) {
	ta := newTestApp(t)
	h := ta.capture("x")
	ta.run("--close", h)
	out, _, code := ta.run("--close", strings.ToUpper(h))
	if code != 0 || strings.TrimSpace(out) != h+" already closed" {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestCloseAllClosesEveryOpenItem(t *testing.T) {
	ta := newTestApp(t)
	ta.capture("one")
	ta.capture("two")
	out, _, code := ta.run("--close", "--all")
	if code != 0 || strings.TrimSpace(out) != "closed 2" {
		t.Fatalf("code %d, out %q", code, out)
	}
	out, _, _ = ta.run("--all", "--close")
	if strings.TrimSpace(out) != "nothing open" {
		t.Errorf("second --close --all: %q", out)
	}
}

func TestUsageErrorsExitTwo(t *testing.T) {
	for _, args := range [][]string{
		{"--clsoe", "k7f"}, // typo must not be captured
		{"--close", "--show", "k7f"},
		{"--close"},
		{"--close", "a", "b"},
		{"--close", "--all", "a"},
		{"--show"},
		{"--edit", "k7f"},
		{""},
		{"--all"},
	} {
		ta := newTestApp(t)
		if _, errOut, code := ta.run(args...); code != 2 || errOut == "" {
			t.Errorf("%q: code %d, stderr %q; want 2", args, code, errOut)
		}
		if view, _, _ := ta.run(); strings.TrimSpace(view) != "nothing open" {
			t.Errorf("%q captured something:\n%s", args, view)
		}
	}
}

func TestHelpAndVersionExitZero(t *testing.T) {
	for _, flag := range []string{"--help", "-h", "--version"} {
		if out, _, code := newTestApp(t).run(flag); code != 0 || out == "" {
			t.Errorf("%s: code %d, out %q", flag, code, out)
		}
	}
}

func TestEditReplacesText(t *testing.T) {
	ta := newTestApp(t)
	h := ta.capture("look into the flaky logn test")
	out, _, code := ta.run("--edit", h, "look", "into", "the", "flaky", "login", "test")
	if code != 0 || strings.TrimSpace(out) != h+"  look into the flaky login test" {
		t.Fatalf("edit: code %d, out %q", code, out)
	}
	if view, _, _ := ta.run(); !strings.Contains(view, "○ look into the flaky login test") {
		t.Errorf("view after edit:\n%s", view)
	}
	if _, errOut, code := ta.run("--edit", "zzz", "x"); code != 1 || !strings.Contains(errOut, "no item zzz") {
		t.Errorf("edit unknown: code %d, stderr %q", code, errOut)
	}
}

func TestShowPrintsEveryField(t *testing.T) {
	ta := newTestApp(t)
	h := ta.capture("retry logic")
	out, _, code := ta.run("--show", h)
	for _, want := range []string{"hash: " + h, "text: retry logic", "state: open", "cwd: " + ta.app.cwd} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("--show missing %q:\n%s", want, out)
		}
	}
	if code != 0 {
		t.Errorf("code %d", code)
	}
}

// runJSON runs aaa with --json and decodes stdout into v.
func (ta *testApp) runJSON(v any, args ...string) int {
	ta.t.Helper()
	out, errOut, code := ta.run(append(args, "--json")...)
	if code == 0 {
		if err := json.Unmarshal([]byte(out), v); err != nil {
			ta.t.Fatalf("%q: bad JSON %q: %v", args, out, err)
		}
	} else if err := json.Unmarshal([]byte(errOut), v); err != nil {
		ta.t.Fatalf("%q: bad JSON error %q: %v", args, errOut, err)
	}
	return code
}

func TestJSONEverywhere(t *testing.T) {
	ta := newTestApp(t)
	var captured map[string]any
	ta.runJSON(&captured, "fix", "the", "thing")
	h, _ := captured["hash"].(string)
	if captured["text"] != "fix the thing" || captured["state"] != "open" || len(h) != 3 {
		t.Fatalf("capture JSON: %v", captured)
	}
	ta.capture("second")

	var view struct {
		Items []struct {
			Position     int    `json:"position"`
			Hash         string `json:"hash"`
			CommitsSince *int   `json:"commits_since"`
		} `json:"items"`
		Open   int `json:"open"`
		Closed int `json:"closed"`
	}
	ta.runJSON(&view)
	if view.Open != 2 || len(view.Items) != 2 || view.Items[0].Position != 1 || view.Items[0].Hash != h ||
		view.Items[1].Position != 2 || view.Items[0].CommitsSince != nil {
		t.Errorf("view JSON: %+v", view)
	}

	var closed map[string]any
	ta.runJSON(&closed, "--close", h)
	if closed["state"] != "closed" || closed["closed_at"] == nil {
		t.Errorf("close JSON: %v", closed)
	}

	var all struct {
		Closed []map[string]any `json:"closed"`
	}
	ta.runJSON(&all, "--close", "--all")
	if len(all.Closed) != 1 || all.Closed[0]["text"] != "second" {
		t.Errorf("close --all JSON: %+v", all)
	}

	var failure map[string]string
	if code := ta.runJSON(&failure, "--show", "zzz"); code != 1 || failure["error"] != "no item zzz" {
		t.Errorf("error JSON: code %d, %v", code, failure)
	}
	if code := ta.runJSON(&failure, "--bogus"); code != 2 || failure["error"] == "" {
		t.Errorf("usage error JSON: code %d, %v", code, failure)
	}
}

func (ta *testApp) at(day, hour, min int) {
	ta.app.now = func() time.Time { return time.Date(2026, 9, day, hour, min, 0, 0, time.Local) }
}

func TestViewShowsAllOpenButOnlyTodaysClosed(t *testing.T) {
	ta := newTestApp(t)
	ta.at(28, 10, 0)
	oldOpen := ta.capture("old open")
	oldClosed := ta.capture("old closed")
	ta.run("--close", oldClosed)

	ta.at(30, 8, 0)
	closedToday := ta.capture("closed today")
	ta.at(30, 9, 0)
	ta.capture("new open")
	ta.at(30, 9, 30)
	ta.run("--close", closedToday)

	view, _, _ := ta.run()
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	want := []string{
		"  1  " + closedToday + "  ✓ closed today",
		"  2  " + oldOpen + "  ○ old open",
		"  3  ",
	}
	if len(lines) != 5 {
		t.Fatalf("want 3 items + blank + footer, got:\n%s", view)
	}
	for i, w := range want {
		if !strings.HasPrefix(lines[i], w) {
			t.Errorf("line %d = %q, want prefix %q", i+1, lines[i], w)
		}
	}
	if !strings.Contains(lines[1], "2026-09-28") || !strings.HasSuffix(lines[2], "09:00") {
		t.Errorf("time column: %q / %q", lines[1], lines[2])
	}
	if strings.Contains(view, "old closed") || lines[4] != "2 open, 1 closed" {
		t.Errorf("view:\n%s", view)
	}
}

// git runs git in dir and fails the test on error.
func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestGitContextAndCommitsSince(t *testing.T) {
	ta := newTestApp(t)
	repo := filepath.Join(t.TempDir(), "example-app")
	os.Mkdir(repo, 0o755)
	git(t, repo, "init", "-q", "-b", "feat/retry")
	git(t, repo, "commit", "-q", "--allow-empty", "-m", "one")
	ta.app.cwd = repo
	h := ta.capture("retry logic")

	line := func() string {
		view, _, _ := ta.run()
		return strings.Split(view, "\n")[0]
	}
	if l := line(); !strings.Contains(l, "example-app  feat/retry") || !strings.HasSuffix(l, "0 commits since") {
		t.Fatalf("fresh capture: %q", l)
	}

	git(t, repo, "commit", "-q", "--allow-empty", "-m", "two")
	git(t, repo, "commit", "-q", "--allow-empty", "-m", "three")
	if l := line(); !strings.HasSuffix(l, "2 commits since") {
		t.Errorf("after 2 commits: %q", l)
	}

	git(t, repo, "reset", "-q", "--hard", "HEAD~2")
	git(t, repo, "commit", "-q", "--amend", "--allow-empty", "-m", "rewritten")
	if l := line(); strings.Contains(l, "since") {
		t.Errorf("rewritten history should show no count: %q", l)
	}

	os.RemoveAll(repo)
	if l := line(); strings.Contains(l, "since") || !strings.Contains(l, h) {
		t.Errorf("deleted repo should show no count: %q", l)
	}
}

func TestEnvAllowlistCapturesAgentVarsButNotSecrets(t *testing.T) {
	ta := newTestApp(t)
	ta.app.env["AI_AGENT"] = "claude-code_2-1-287_agent"
	ta.app.env["CLAUDE_CODE_SESSION_ID"] = "00000000"
	ta.app.env["PI_SESSION_ID"] = "11111111"
	ta.app.env["CLAUDE_CODE_MESSAGING_TOKEN"] = "secret"
	ta.app.env["JIRA_API_TOKEN"] = "secret"
	h := ta.capture("x")

	var it struct {
		Env map[string]string `json:"env"`
	}
	ta.runJSON(&it, "--show", h)
	want := map[string]string{"AI_AGENT": "claude-code_2-1-287_agent", "CLAUDE_CODE_SESSION_ID": "00000000", "PI_SESSION_ID": "11111111"}
	if !maps.Equal(it.Env, want) {
		t.Errorf("env = %v, want %v", it.Env, want)
	}
}

// fakeCommand puts an executable shell script named name first on PATH.
func fakeCommand(t *testing.T, name, script string) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestHerdrAndTmuxLabels(t *testing.T) {
	fakeCommand(t, "herdr", `case "$1 $2" in
"workspace list") echo '{"id":"cli:workspace:list","result":{"type":"workspace_list","workspaces":[{"workspace_id":"w5W","label":"other"},{"workspace_id":"w6B","label":"aaa"}]}}' ;;
"tab list") echo '{"id":"cli:tab:list","result":{"type":"tab_list","tabs":[{"tab_id":"w6B:t1","label":"1"},{"tab_id":"w6B:t2","label":"build"}]}}' ;;
esac`)
	fakeCommand(t, "tmux", `echo "work:editor"`)
	ta := newTestApp(t)
	for k, v := range map[string]string{"HERDR_ENV": "1", "HERDR_WORKSPACE_ID": "w6B", "HERDR_TAB_ID": "w6B:t2", "TMUX": "/tmp/tmux-1000/default,1,0"} {
		ta.app.env[k] = v
	}
	h := ta.capture("x")
	out, _, _ := ta.run("--show", h)
	for _, want := range []string{"herdr_workspace: aaa\n", "herdr_tab: build\n", "tmux: work:editor\n", "env.HERDR_TAB_ID: w6B:t2\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestHashUsesSafeAlphabetAndGrowsWhenCrowded(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 1000 {
		h := newHash(func(string) bool { return false }, r)
		if len(h) != 3 || strings.Trim(h, "23456789abcdefghijkmnpqrstuvwxyz") != "" {
			t.Fatalf("bad hash %q", h)
		}
	}
	allThreeTaken := func(h string) bool { return len(h) == 3 }
	if h := newHash(allThreeTaken, r); len(h) != 4 {
		t.Errorf("crowded: got %q, want 4 characters", h)
	}
}

func TestSkillPrintsSkillFile(t *testing.T) {
	out, _, code := newTestApp(t).run("--skill")
	if code != 0 || !strings.HasPrefix(out, "---\nname: aaa\n") {
		t.Fatalf("code %d, out starts %q", code, out[:min(len(out), 40)])
	}
	for _, want := range []string{"triple A", "AAA", "triple-A", "3A", "aaa --json", "--close --all", "--edit"} {
		if !strings.Contains(out, want) {
			t.Errorf("skill text missing %q", want)
		}
	}
}
