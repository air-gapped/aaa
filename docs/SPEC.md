# AAA v1 — specification

Items marked **(default)** were not discussed with the user; they are the spec's choice and can be changed.

## 1. Purpose

`aaa` records "I started this" with zero friction. It is a memory aid, not a task tracker.
It does not integrate with Beads, Jira or GitLab, and it has no export.

## 2. Build and storage

- Go, one static binary, `CGO_ENABLED=0`. It runs air-gapped with nothing else installed.
- SQLite through a pure-Go driver, `modernc.org/sqlite` **(default)**. It is the only third-party dependency.
- Database: `$XDG_DATA_HOME/aaa/aaa.db`, default `~/.local/share/aaa/aaa.db`.
- The first run creates the directory and the database silently. There is no `init` command.
- Schema version lives in `PRAGMA user_version`. Migrations run on open.
- `PRAGMA journal_mode=WAL` and `busy_timeout=5000`, because several agents can write at the same time **(default)**.
- Single machine. No sync.

## 3. Command grammar

There are no subcommands and no reserved words. Bare words are capture text. Operations are flags.

| Invocation | Effect |
|---|---|
| `aaa` | Show the view (§6). |
| `aaa <words…>` | Capture one item. The words are joined with single spaces. Quotes are optional. |
| `aaa --close <hash>` | Close one item. |
| `aaa --close --all` | Close every open item. |
| `aaa --edit <hash> <words…>` | Replace the item's text. |
| `aaa --show <hash>` | Print one item in full. |
| `aaa --skill` | Print the agent skill text (§8). |
| `aaa --help`, `-h` | Print usage. |
| `aaa --version` | Print the version. |

`--json` combines with every row above, except `--skill`, `--help` and `--version`, and changes the output to JSON (§7). Combining it with those three is a usage error (exit 2).

Parsing rules:

1. An argument is a flag only when it exactly matches a known flag.
2. An argument that starts with `-` and is not a known flag is a usage error (exit 2). This stops a typo such as `aaa --clsoe k7f` from being captured as an item **(default)**.
3. Flags may appear anywhere. All other arguments are positional.
4. `--close`, `--edit` and `--show` take the first positional as the hash. `--edit` takes the remaining positionals as the new text. `--close` and `--show` accept exactly one positional, or with `--close`, `--all` and none.
5. Only one operation flag per call (`--close`, `--edit`, `--show`, `--skill`, `--help`, `--version`). Two or more is a usage error.
6. With no operation flag and no positionals: show the view. With positionals: capture.
7. Empty capture text (for example `aaa ""`) is a usage error.

Hashes are matched case-insensitively.

## 4. Captured metadata

At capture time `aaa` stores the following. Every probe is best-effort: a failed or missing probe leaves the field empty and never fails the capture. Each external command has a 1-second timeout **(default)**.

| Field | Source |
|---|---|
| `text` | the arguments |
| `created_at` | current time, stored UTC, RFC 3339 |
| `cwd` | working directory |
| `repo` | `git rev-parse --show-toplevel` |
| `branch` | `git rev-parse --abbrev-ref HEAD` (`HEAD` when detached) |
| `sha` | `git rev-parse HEAD` |
| `tmux` | `tmux display-message -p '#S:#W'`, only when `TMUX` is set |
| `tty` | the terminal on stdin (`/proc/self/fd/0` link), only when stdin is a terminal |
| `herdr_workspace`, `herdr_tab` | workspace label and tab label from `herdr workspace list` and `herdr tab list`, only when `HERDR_ENV=1` |
| `env` | every variable from the allowlist below that is set, verbatim, as a JSON object of name → value |

Git is called as the `git` binary. If `git` is not installed, the git fields stay empty.

### Environment allowlist

Exact names only. Never match by prefix: agent environments also hold secrets (for example `CLAUDE_CODE_MESSAGING_TOKEN`, API tokens).
Do not derive a `source` field from these. Store them as they are.

```
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
```

Seen live on 2026-10-01: Claude Code, OpenCode v2.0.20 and Pi. The others come from research (codex-rs source, READMEs) and are unverified.
Adding a name is a one-line change to this list.

Not stored: hostname, dirty-worktree flag.

## 5. IDs

- Each item has a 3-character hash from the alphabet `23456789abcdefghijkmnpqrstuvwxyz`, which leaves out 0, 1, o and l. That gives 32 characters and 32,768 hashes.
- The hash is random and unique across all items, open and closed, forever. On a collision `aaa` draws again. After 10 collisions in a row it uses 4 characters **(default)**.
- There are no position numbers. They renumber on every listing, so "close two" could hit a different item if anything changed in between. By voice, the user says the hash or describes the item, and the agent matches the description.

## 6. View (bare `aaa`)

- Shows every open item, from any day, and the items closed today (local calendar day).
- One flat list. No grouping and no ✓/○ marks.
- Order: items closed today first, by `closed_at`; then open items by `created_at`, oldest first. The newest open item is on the last line, right above the prompt.
- Items closed today are drawn dimmed. Nothing else marks them.
- A dimmed header row: `ID  STARTED  TEXT  GIT  PUSH  PULL  PATH`.
- Empty cells stay blank. No `—`.
- Footer, dimmed: `N open, M done today`.
- Empty state: one line, `nothing open`.
- No truncation and no ranking. 200 open items print 200 lines.

Columns:

| Column | Content |
|---|---|
| ID | the hash, yellow |
| STARTED | clock time `HH:MM`, dimmed. Weekday and date (`Thu Sep 17`) are shown only when the item's day differs from the line above. Today follows the same rule, so its first line shows today's date: every bare time belongs to the dated line above it. Weekdays line up under weekdays; clock times line up in their own column. Never relative ("5m", "3d"). |
| TEXT | the item text |
| GIT | branch name only, cyan. The repo is not repeated: PATH shows it. |
| PUSH | number of local commits not on the remote branch, green, right-aligned. Blank when 0. |
| PULL | `yes` when the remote branch has a commit we do not have, red, right-aligned. Blank otherwise. |
| PATH | the working directory at capture, home shortened to `~`, dimmed |

PUSH and PULL are computed live with `git ls-remote <remote> refs/heads/<branch>`, which asks the server for the branch's commit ID and downloads nothing (measured 0.46 s over HTTPS, 1.4 s over SSH). Repos are checked in parallel. PULL has no count, because counting would need a fetch. Login prompts must fail quietly (`GIT_TERMINAL_PROMPT=0`, SSH `BatchMode=yes`). An unreachable server or a branch without a remote leaves both cells blank. This is never an error.

Commits since capture (`git rev-list --count <sha>..HEAD`) is not shown in the list. It stays in `--show` and in the JSON as `commits_since`.

```
$ aaa
  ID   STARTED           TEXT                            GIT         PUSH  PULL  PATH
  2em  Thu Oct 1  09:14  update the README                                       ~/src/example-app
  k4p  Thu Sep 17 10:41  look into the flaky login test  main               yes  ~/src/example-app/tests
  m7c             15:22  check why the build is slow                             ~
  x2m  Mon Sep 28 16:03  finish the config cleanup       main                    ~/src/example-lib
  yv3  Thu Oct 1  21:13  add retry to the upload job     feat/retry     2        ~/src/example-app/upload

  4 open, 1 done today
```

## 7. Operations and output

| Operation | Text output | Exit |
|---|---|---|
| capture | `k7f  <text>` | 0 |
| `--close <hash>` | `closed k7f  <text>` | 0 |
| `--close <hash>`, already closed | `k7f already closed` | 0 |
| `--close <unknown>` | `aaa: no item zzz` on stderr | 1 |
| `--close --all` | `closed N` (`nothing open` when N is 0) | 0 |
| `--edit <hash> <text>` | `k7f  <new text>`; closed items can be edited too | 0 |
| `--edit` or `--show` with an unknown hash | `aaa: no item zzz` on stderr | 1 |
| `--show <hash>` | every stored field, one `name: value` per line | 0 |
| usage error | message and a short usage hint on stderr | 2 |
| database error | message on stderr | 1 |

Closing sets `closed_at`. There is one closed state; there is no reopen in v1.

### JSON

With `--json`, every success prints one JSON document to stdout. Errors print `{"error": "<message>"}` to stderr, with the same exit codes.

Item object (used by capture, `--close`, `--edit`, `--show`):

```json
{
  "hash": "x2m",
  "text": "add retry to the upload job",
  "state": "open",
  "created_at": "2026-09-29T08:27:03Z",
  "closed_at": null,
  "cwd": "/home/user/src/example-app",
  "repo": "/home/user/src/example-app",
  "branch": "feat/retry",
  "sha": "9f2c…",
  "tmux": "",
  "tty": "/dev/pts/4",
  "herdr_workspace": "work",
  "herdr_tab": "1",
  "env": { "AI_AGENT": "claude-code_2-1-287_agent", "CLAUDE_CODE_SESSION_ID": "00000000-0000-0000-0000-000000000000" }
}
```

The view (`aaa --json`) prints `{"items": [...], "open": 3, "closed": 2}`. Each item adds `commits_since` (integer, or `null` when it cannot be computed), `push` (integer, `null` when unknown) and `pull` (boolean, `null` when unknown), the same remote state as the PUSH and PULL columns.
`--close --all --json` prints `{"closed": [<item>, ...]}`.

## 8. `--skill`

`aaa --skill` prints a complete `SKILL.md`, with YAML frontmatter, to stdout. The text is embedded in the binary with `go:embed`. Installing it is `aaa --skill > ~/.claude/skills/aaa/SKILL.md`. No other artifact exists.

The skill text must say:

- The tool is named by the user. Spoken and transcribed variants: "triple A", "AAA", "triple-A", "3A", "aaa". There is no other trigger phrase.
- Capture: `aaa --json <text>`. The agent may capture by itself, for example for work left unfinished at the end of a session.
- Read: `aaa --json`. Never parse the table output.
- When the user describes an item instead of giving its hash ("close the README one"), match the description against the most recent `aaa --json` listing, then run one `aaa --close <hash> --json` per item. If the match is not clear, ask.
- "Close them all": `aaa --close --all --json`.
- Close only when the user says so. Never close on the agent's own judgement.
- Fix dictation errors with `aaa --edit <hash> <text> --json`.
- AAA is not a tracker. If an item needs a ticket, create it in the right tool (Beads, Jira, GitLab); AAA has no export.

## 9. Code layout

One Go module, one `package main`, few files **(default)**:

| File | Responsibility |
|---|---|
| `main.go` | argument parsing, dispatch, exit codes |
| `store.go` | open, migrate, insert, close, edit, query |
| `probe.go` | git, tmux, tty, herdr, environment allowlist |
| `view.go` | text and JSON rendering, commits-since |
| `hash.go` | hash generation |
| `skill.md` | embedded skill text |

## 10. Tests (test-first)

- Grammar: bare words capture; quotes are optional; each flag form; flag position anywhere; unknown `--x` exits 2; two operation flags exit 2; empty text exits 2.
- Hash: only alphabet characters; uniqueness under forced collisions; growth to 4 characters.
- Store: close, close again, close unknown, `--all`, edit; database created on first run in a temp `XDG_DATA_HOME`.
- View: open items from earlier days are included, closed items only from today; ordering; empty state.
- Commits-since against a temp git repo: a normal count, a rebased SHA (blank), a deleted repo (blank).
- Environment allowlist: listed names are captured; an unlisted secret such as `CLAUDE_CODE_MESSAGING_TOKEN` is not.

## 11. Out of scope for v1

Parent-child links (`--under`), separate done/drop states, reopen, TUI, yak stack, scheduled reports, a `day` command, idempotency keys, `--since`, tracker export, nagging, sync.
