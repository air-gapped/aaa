package main

import (
	"context"
	"encoding/json/v2"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// probe runs a command in dir with a 1-second timeout. Probes are
// best-effort: a failure only means the field stays empty.
func probe(dir, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// output returns the command's trimmed stdout, or "" on any failure.
func output(dir, name string, args ...string) string {
	out, err := probe(dir, name, args...)
	if err != nil {
		return ""
	}
	return out
}

// probeGit fills repo, branch and sha when cwd is inside a git repo.
func probeGit(it *item) {
	it.Repo = output(it.Cwd, "git", "rev-parse", "--show-toplevel")
	if it.Repo == "" {
		return
	}
	it.Branch = output(it.Cwd, "git", "rev-parse", "--abbrev-ref", "HEAD")
	it.SHA = output(it.Cwd, "git", "rev-parse", "HEAD")
}

// commitsSince counts commits on HEAD after sha, or returns nil when the
// repo is gone or sha is no longer an ancestor of HEAD.
func commitsSince(it *item) *int {
	if it.Repo == "" || it.SHA == "" {
		return nil
	}
	n, err := strconv.Atoi(output(it.Repo, "git", "rev-list", "--count", it.SHA+"..HEAD"))
	if err != nil {
		return nil
	}
	if _, err := probe(it.Repo, "git", "merge-base", "--is-ancestor", it.SHA, "HEAD"); err != nil {
		return nil
	}
	return &n
}

// envAllowlist names the variables captured verbatim. Exact names only:
// agent environments also carry secrets, so never match by prefix.
var envAllowlist = strings.Fields(`
	AI_AGENT AGENT
	CLAUDECODE CLAUDE_CODE_SESSION_ID CLAUDE_CODE_ENTRYPOINT
	OPENCODE OPENCODE_SESSION_ID OPENCODE_CLIENT
	PI_CODING_AGENT PI_SESSION_ID PI_SESSION_FILE PI_PROVIDER PI_MODEL
	CODEX_THREAD_ID CODEX_SESSION_ID CODEX_VERSION CODEX_SANDBOX
	COPILOT_AGENT COPILOT_AGENT_SESSION_ID COPILOT_AGENT_JOB_ID
	GEMINI_CLI
	CURSOR_AGENT CURSOR_TRACE_ID
	CLINE_ACTIVE CLINE_TASK_ID
	GOOSE_TERMINAL AGENT_SESSION_ID
	AMP_CURRENT_THREAD_ID AGENT_THREAD_ID
	QWEN_CODE_SESSION_ID
	HERDR_WORKSPACE_ID HERDR_TAB_ID HERDR_PANE_ID
	TMUX TMUX_PANE
`)

// probeSession fills the terminal context: tty, tmux, herdr and the
// allowlisted environment.
func probeSession(it *item, env map[string]string) {
	it.Env = map[string]string{}
	for _, k := range envAllowlist {
		if v, ok := env[k]; ok {
			it.Env[k] = v
		}
	}
	if tty, err := os.Readlink("/proc/self/fd/0"); err == nil &&
		(strings.HasPrefix(tty, "/dev/pts/") || strings.HasPrefix(tty, "/dev/tty")) {
		it.TTY = tty
	}
	if env["TMUX"] != "" {
		it.Tmux = output(it.Cwd, "tmux", "display-message", "-p", "#S:#W")
	}
	if env["HERDR_ENV"] == "1" {
		it.HerdrWorkspace = herdrLabel(it.Cwd, "workspaces", "workspace_id", env["HERDR_WORKSPACE_ID"],
			"workspace", "list")
		it.HerdrTab = herdrLabel(it.Cwd, "tabs", "tab_id", env["HERDR_TAB_ID"],
			"tab", "list", "--workspace", env["HERDR_WORKSPACE_ID"])
	}
}

// herdrLabel runs a herdr list command and returns the label of the
// entry whose idField equals id.
func herdrLabel(dir, list, idField, id string, args ...string) string {
	if id == "" {
		return ""
	}
	var resp struct {
		Result map[string]any `json:"result"`
	}
	if json.Unmarshal([]byte(output(dir, "herdr", args...)), &resp) != nil {
		return ""
	}
	entries, _ := resp.Result[list].([]any)
	for _, e := range entries {
		if e, _ := e.(map[string]any); e[idField] == id {
			label, _ := e["label"].(string)
			return label
		}
	}
	return ""
}
