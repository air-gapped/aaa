# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] (unreleased)

The first release.

### Added

- Capture: `aaa <words>` records what you started, with the working
  directory, git repo, branch and commit, terminal, tmux and herdr context,
  and an allowlist of coding-agent environment variables (session IDs for
  Claude Code, OpenCode, Pi, Codex and others).
- List: bare `aaa` shows every open item and the items closed today, oldest
  first. Dates appear only when the day changes. GIT shows the branch;
  PUSH and PULL show unpushed commits and new commits on the server, checked
  live with `git ls-remote` and blank when there is nothing to do.
- `--close <hash>`, `--close --all`, `--edit <hash> <text>`, `--show <hash>`.
- `--json` on every operation, for coding agents.
- `--skill` prints the agent skill, so the binary is the only artifact.
