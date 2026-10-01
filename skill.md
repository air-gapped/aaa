---
name: aaa
description: "Capture, list, edit and close 'I started this' items with the aaa CLI. Use when the user names the tool: \"triple A\", \"AAA\", \"triple-A\", \"3A\" or \"aaa\" (voice transcription produces all of these), for example \"put that in triple A\", \"what's in AAA\", \"close one and three in triple A\". Also use at the end of a session to record work left unfinished."
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

Every word that is not a flag is capture text. Quotes are optional. A word that starts with `-` and is not a known flag is an error (exit 2), so rephrase text that starts with a dash.

## Hashes and positions

Each item has a 3-character hash, for example `k7f`. Commands take hashes only.

The user sees a position number (1, 2, 3 …) in front of each item and may say "close one, two and four". Positions are not accepted by `aaa`. Map them to hashes from the `position` field of the most recent `aaa --json` listing, then run one `aaa --close <hash> --json` per item. If you have no recent listing, run `aaa --json` first.

## Rules

- The user names the tool. There is no other trigger phrase.
- You may capture by yourself, for example work left unfinished at the end of a session. Write the text so the user recognises it later.
- Close items only when the user tells you to. Never close an item on your own judgement, even if the work looks done.
- "Close them all" means `aaa --close --all --json`.
- Dictation errors: fix them with `--edit`. Do not close and recapture.
- If an item needs a real ticket, create it in the right tool (Beads, Jira, GitLab). `aaa` has no export.

## Exit codes

- 0: success. Closing an item that is already closed is also 0.
- 1: unknown hash or database error. The JSON error is on stderr: `{"error": "no item zzz"}`.
- 2: usage error.
