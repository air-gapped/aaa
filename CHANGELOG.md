# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0](https://github.com/air-gapped/aaa/commits/v0.1.0) (2026-10-02)

The first release. `aaa` ("triple A") writes down what you started, in one
command, so an interruption does not make you forget it. Linux only, amd64 and
arm64, one static binary with nothing else to install.

### Highlights

#### Capture In One Command

`aaa fix the flaky login test` stores the text with where you were: the
directory, the git repo, branch and commit, the terminal, the tmux and herdr
labels, and the session of the coding agent that ran it (Claude Code,
OpenCode, Pi, Codex and others, read from an exact list of environment
variables; secrets such as tokens are never stored). No quotes needed. Text is
limited to 1000 characters.

#### A List You Can Read Tired

Bare `aaa` shows every open item and what you closed today, oldest first, with
a dimmed header. Dates appear only where the day changes. GIT shows the
branch; PUSH counts commits you have not pushed and PULL says `yes` when the
server has something new, both checked live and blank when there is nothing to
do. Items closed today are dimmed.

#### Built For Coding Agents

Every operation takes `--json`. `aaa --skill` prints the agent skill that
matches this exact binary, so installing it is one line:
`aaa --skill > ~/.claude/skills/aaa/SKILL.md`. Agents close items by their
3-character ID, or by matching your description, and only when you ask.

### Features

* capture, list, `--close`, `--close --all`, `--edit` and `--show`, with `--json` on every operation ([ea35555](https://github.com/air-gapped/aaa/commit/ea35555f980411e6e749999a9dd8e18955c579fa))
* the list shows live PUSH and PULL state, dates only when the day changes, and colour only on a terminal (`NO_COLOR` turns it off) ([c234f72](https://github.com/air-gapped/aaa/commit/c234f7256b42a00603bac79c8ca9cd35fa6901d5))
* item text is limited to 1000 characters ([1bcaf5e](https://github.com/air-gapped/aaa/commit/1bcaf5e0c0ed55e6443ff2597a0307e856401c8c))
* capturing from a directory that has been deleted records the shell's `$PWD` instead of an empty path ([b6a389b](https://github.com/air-gapped/aaa/commit/b6a389b2356eb31de31df7b9843d993d0aa17918))

### Security

* a directory name or other stored text containing terminal escape sequences is shown as visible `\xNN` and can no longer drive your terminal ([e155039](https://github.com/air-gapped/aaa/commit/e15503919d43d935ea57ba1b368c615021d49638))
