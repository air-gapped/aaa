---
name: release
description: >-
  Release workflow for aaa — the agent runs the whole release: version
  choice, changelog section, gate, dry run, release run. Use when preparing a
  release, writing or reviewing release notes, checking whether a release is
  ready, fixing a release that went wrong, or understanding how aaa versions
  work.
---

# Release workflow

**The agent runs the release end to end.** There is no release bot, and there
never will be one in this repo (no release-please). A release is a git tag plus
a published GitHub release with Linux amd64 and arm64 archives. Pushing `main`
does **not** start one.

## The sequence

The order matters: a tag is a public claim and must never point at a commit
that failed a check.

```bash
# 1. Everything intended for the release is committed and on main.
git status --short                      # clean
git log --oneline <last-tag>..HEAD      # first release: git log --oneline

# 2. Choose the version from what is IN that range, not from habit.
#    New capability -> minor. Fix only -> patch. A changed or removed --json
#    field, flag, exit code or skill command is BREAKING: minor while 0.x,
#    major after 1.0.

# 3. Write the CHANGELOG.md section BY HAND (format below), date it, commit.
git commit -am "chore: release X.Y.Z"

# 4. The gate. This decides "ready", not judgement.
make ci                                 # gofmt, vet, tests, govulncheck
make fuzz FUZZTIME=1m                   # parser and terminal sanitizer
make mutants                            # any new LIVED mutant needs a test
#    The one accepted survivor is the TTY check (probe.go, depends on the
#    real terminal). Anything else that lives is untested behaviour.
#    If list output changed since the last release: make demo, check the
#    GIF and the README examples against real output.

# 5. Dry run: both targets tested, built, smoke-tested, packaged; nothing
#    tagged or published.
git push origin main
gh workflow run release.yml --ref main
gh run watch "$(gh run list --workflow release.yml --limit 1 --json databaseId -q '.[0].databaseId')" --exit-status

# 6. The release, only after the dry run passed. The same jobs run again, and
#    only if every one is green does the workflow create the tag and the
#    GitHub release at the tested commit, with the notes cut from the
#    version's CHANGELOG.md section.
gh workflow run release.yml --ref main -f tag=vX.Y.Z
gh run watch "$(gh run list --workflow release.yml --limit 1 --json databaseId -q '.[0].databaseId')" --exit-status

# 7. Check what users get.
gh release view vX.Y.Z
gh release download vX.Y.Z --pattern '*linux-amd64.tar.gz' --dir /tmp/aaa-check
tar -xzf /tmp/aaa-check/*.tar.gz -C /tmp/aaa-check && /tmp/aaa-check/aaa --version   # prints vX.Y.Z
```

Never tag by hand and never run `gh release create` by hand: the workflow
creates both, after every target has passed.

## Writing the release notes

The notes are the version's `CHANGELOG.md` section; the workflow publishes it
verbatim. **Users read this. Say what changed for them, not what moved in the
code.** Write each line from the reader's side: what they type, what they see,
what they must change.

```markdown
## [X.Y.Z](https://github.com/air-gapped/aaa/compare/vA.B.C...vX.Y.Z) (YYYY-MM-DD)

One short paragraph: what this release is, and whether upgrading needs any
action. If nothing breaks, say so.

### Highlights

#### <Title Case, one per user-visible theme>

What the user can now do or will now see, with the exact flags, defaults and
limits. Name what is deliberately left out or capped, so nothing surprises.

#### Upgrading From A.B.C

Only when something breaks: each change, and the exact way to get the old
behaviour back.

### ⚠ BREAKING CHANGES

* one line per break, in user terms

### Features

* one line per change, user terms ([abc1234](https://github.com/air-gapped/aaa/commit/<sha>))

### Bug Fixes

* ...

### Security

* ...
```

Rules:

- Lead with behaviour, never with implementation: "a directory name can no
  longer drive your terminal", not "added clean() in view.go".
- Every limit, default and cap gets its number (1000 characters, 5 s, 2 min).
- One line per item in the lists; the Highlights carry the explanation.
- Omit empty sections. A first release has no "Upgrading" and no breaking list.
- The heading must start with `## [X.Y.Z]`: the workflow's plan job refuses a
  tag whose section is missing, and cuts the notes at the next `## [`.

## Version rules

- Go has no version file. **The tag is the version**: the release workflow
  stamps it into the binary with `-X main.version=<tag>`; local builds stamp
  `git describe` (`make build`).
- Tags are `vX.Y.Z`. A tag is public the moment the workflow pushes it; never
  push one speculatively, and never move one.

## If something goes wrong

- **A test, build or smoke test fails in the release run:** nothing is
  public; the tag is created only after both targets pass. Fix on main, push,
  run the release again.
- **The release was created but is broken:** turn it back into a draft
  (`gh release edit vX.Y.Z --draft`), fix on main, and release the next patch
  version. A published tag is never reused or moved.
