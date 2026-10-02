---
name: aaa
description: "Capture, list, edit and close 'I started this' items with the aaa CLI. Use when the user names the tool: \"triple A\", \"AAA\", \"triple-A\", \"3A\" or \"aaa\" (voice transcription produces all of these), for example \"put that in triple A\", \"what's in AAA\", \"close the README one in triple A\". Also use at the end of a session to record work left unfinished."
---

# aaa

`aaa` is a memory aid: it records what the user started so it is not lost when they get interrupted. It is not a task tracker.

## Commands

Always pass `--json` and parse the JSON. Never parse the table output.

| Task | Command |
|---|---|
| Capture an item | `aaa --json <text>` |
| List open items and today's closed items | `aaa --json` |
| Close one item | `aaa --close <hash> --json` |
| Close every open item | `aaa --close --all --json` |
| Fix an item's text | `aaa --edit <hash> <new text> --json` |
| Show one item in full | `aaa --show <hash> --json` |

## Hashes

Each item has a 3-character hash, for example `k7f`. Commands take hashes only.

When the user describes an item instead of giving its hash ("close the README one"), match the description against the `text` of the items in the most recent `aaa --json` listing, then run one `aaa --close <hash> --json` per item. If the match is not clear, or several items fit, ask. With no recent listing, run `aaa --json` first.

## Rules

- Capture unprompted only for work left unfinished at the end of a session. Write the text so the user recognises it later.
- Close an item only on the user's instruction, even when the work looks done.
- "Close them all" means `aaa --close --all --json`.
- "What's in AAA?" means: run `aaa --json` and answer with one line per open item, `hash  text`, oldest first. Mention push or pull only where `push > 0` or `pull` is `true`; those are the items that need action.
- Dictation errors: fix them with `--edit`. Do not close and recapture.
- If an item needs a real ticket, create it in the right tool (Beads, Jira, GitLab). `aaa` has no export.

## Output and errors

Success prints one JSON document on stdout. Every item carries `hash`, `text`, `state` (`open` or `closed`), `created_at`, `closed_at`, `cwd`, `repo` and `branch`. The listing is `{"items": [...], "open": N, "closed": M}`, and its items add `push` (local commits the server lacks), `pull` (`true` when the server has commits the local branch lacks) and `commits_since` (commits since capture), each `null` when unknown.

| Result | Meaning | Do |
|---|---|---|
| exit 0 | success; closing an already-closed item is also 0 | report the hash and text back |
| exit 1, `{"error": "no item zzz"}` on stderr | unknown hash | run `aaa --json`, match the item again, retry once |
| exit 1, any other error | database or file problem | report the error text to the user; retrying will not help |
| exit 2 | usage error, usually capture text starting with `-` | rephrase the text without a leading dash |
| `command not found` | `aaa` is not installed | tell the user; keep the item in the conversation instead |
