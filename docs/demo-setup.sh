# Sourced (hidden) by docs/demo.tape: a throwaway home with two example
# repos, so the recording shows no real paths, sessions or agent variables.
unset HERDR_ENV HERDR_WORKSPACE_ID HERDR_TAB_ID HERDR_PANE_ID TMUX TMUX_PANE AI_AGENT AGENT \
      CLAUDECODE CLAUDE_CODE_SESSION_ID CLAUDE_CODE_ENTRYPOINT OPENCODE OPENCODE_SESSION_ID
export HOME="$(mktemp -d)" XDG_DATA_HOME= GIT_AUTHOR_NAME=demo GIT_AUTHOR_EMAIL=demo@example.com \
       GIT_COMMITTER_NAME=demo GIT_COMMITTER_EMAIL=demo@example.com
export PS1='$ '
g() { git -c init.defaultBranch=main -c commit.gpgsign=false "$@" >/dev/null 2>&1; }
mkdir -p ~/remotes ~/src ~/ops
for r in example-app example-lib; do
  g init --bare ~/remotes/$r.git
  g clone ~/remotes/$r.git ~/src/$r
  g -C ~/src/$r commit --allow-empty -m init
  g -C ~/src/$r push origin main
done
mkdir -p ~/src/example-app/tests
# example-app: someone else pushed, so PULL shows "yes"
g clone ~/remotes/example-app.git /tmp/other.$$ && g -C /tmp/other.$$ commit --allow-empty -m theirs && g -C /tmp/other.$$ push origin main && rm -rf /tmp/other.$$
# example-lib: two local commits not pushed, so PUSH shows 2
g -C ~/src/example-lib commit --allow-empty -m one && g -C ~/src/example-lib commit --allow-empty -m two
# Older items, with fixed hashes and dates
seed() { (cd "$1" && aaa "$4" >/dev/null) && sqlite3 "$HOME/.local/share/aaa/aaa.db" \
  "UPDATE items SET hash='$2', created_at='$(date -u -d "@$(date -d "$3" +%s)" +%Y-%m-%dT%H:%M:%SZ)' WHERE text='$4'"; }
seed ~/src/example-app/tests k7f '8 days ago 10:41' 'look into the flaky login test'
seed ~/ops m2c '8 days ago 15:22' 'check why the build is slow'
seed ~/src/example-lib x4p '4 days ago 16:03' 'finish the config cleanup'
cd ~/src/example-app
