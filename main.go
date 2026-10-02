// Command aaa records "I started this" with zero friction.
package main

import (
	_ "embed"
	"encoding/json/v2"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"time"
)

type app struct {
	env            map[string]string
	cwd            string
	now            func() time.Time
	rand           *rand.Rand
	json           bool // set per run from --json
	color          bool // ANSI colours on a terminal
	stdout, stderr io.Writer
}

func main() {
	env := map[string]string{}
	for _, kv := range os.Environ() {
		k, v, _ := strings.Cut(kv, "=")
		env[k] = v
	}
	cwd, _ := os.Getwd()
	fi, _ := os.Stdout.Stat()
	color := fi != nil && fi.Mode()&os.ModeCharDevice != 0 && env["NO_COLOR"] == ""
	a := &app{env: env, cwd: cwd, now: time.Now, color: color, stdout: os.Stdout, stderr: os.Stderr,
		rand: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))}
	os.Exit(a.run(os.Args[1:]))
}

//go:embed skill.md
var skill string

var version = "dev" // set with -ldflags "-X main.version=..."

const usage = `usage:
  aaa                          show open items and today's closed items
  aaa <words...>               capture an item
  aaa --close <hash>           close one item
  aaa --close --all            close every open item
  aaa --edit <hash> <words...> replace an item's text
  aaa --show <hash>            show one item in full
  aaa --skill                  print the agent skill (SKILL.md)
  --json                       JSON output, with any of the above except --skill`

// usageError is reported with exit code 2.
type usageError string

func (e usageError) Error() string { return string(e) }

type command struct {
	op        string // "", "--close", "--edit", "--show", "--skill", "--help", "--version"
	all, json bool
	words     []string
}

// parse turns arguments into a command. Bare words are capture text;
// only exact known flags are flags.
func parse(args []string) (command, error) {
	var c command
	for _, arg := range args {
		switch arg {
		case "--close", "--edit", "--show", "--skill", "--help", "-h", "--version":
			if arg == "-h" {
				arg = "--help"
			}
			if c.op != "" {
				return c, usageError(c.op + " and " + arg + " cannot be combined")
			}
			c.op = arg
		case "--all":
			c.all = true
		case "--json":
			c.json = true
		default:
			if len(arg) > 1 && arg[0] == '-' {
				return c, usageError("unknown flag " + arg)
			}
			c.words = append(c.words, arg)
		}
	}
	n := len(c.words)
	switch {
	case c.all && c.op != "--close":
		return c, usageError("--all only works with --close")
	case c.op == "--close" && c.all && n != 0, c.op == "--close" && !c.all && n != 1:
		return c, usageError("--close takes one hash, or --all")
	case c.json && (c.op == "--skill" || c.op == "--help" || c.op == "--version"):
		return c, usageError("--json does not combine with " + c.op)
	case c.op == "--show" && n != 1:
		return c, usageError("--show takes one hash")
	case c.op == "--edit" && (n < 2 || strings.TrimSpace(strings.Join(c.words[1:], "")) == ""):
		return c, usageError("--edit takes a hash and the new text")
	case c.op == "" && n > 0 && strings.TrimSpace(strings.Join(c.words, "")) == "":
		return c, usageError("empty text")
	}
	return c, nil
}

func (a *app) run(args []string) int {
	c, err := parse(args)
	a.json = slices.Contains(args, "--json") // also for errors parse stopped at
	if err != nil {
		return a.fail(err)
	}
	switch c.op {
	case "--help":
		fmt.Fprintln(a.stdout, usage)
		return 0
	case "--version":
		fmt.Fprintln(a.stdout, "aaa", version)
		return 0
	case "--skill":
		fmt.Fprint(a.stdout, skill)
		return 0
	}

	s, err := openStore(a.env)
	if err != nil {
		return a.fail(err)
	}
	defer s.db.Close()

	switch {
	case c.op == "--close" && c.all:
		items, err := s.closeAll(a.now())
		if err != nil {
			return a.fail(err)
		}
		if len(items) == 0 {
			return a.print(map[string]any{"closed": items}, "nothing open\n")
		}
		return a.print(map[string]any{"closed": items}, "closed %d\n", len(items))

	case c.op == "--close":
		it, closed, err := s.close(c.words[0], a.now())
		if err != nil {
			return a.fail(err)
		}
		if !closed {
			return a.print(it, "%s already closed\n", it.Hash)
		}
		return a.print(it, "closed %s  %s\n", it.Hash, it.Text)

	case c.op == "--edit":
		it, err := s.edit(c.words[0], strings.Join(c.words[1:], " "))
		if err != nil {
			return a.fail(err)
		}
		return a.print(it, "%s  %s\n", it.Hash, it.Text)

	case c.op == "--show":
		it, err := s.get(c.words[0])
		if err != nil {
			return a.fail(err)
		}
		if a.json {
			return a.print(it, "")
		}
		showItem(a.stdout, it)
		return 0

	case len(c.words) == 0:
		items, err := a.viewItems(s)
		if err != nil {
			return a.fail(err)
		}
		if a.json {
			return a.print(a.viewJSON(items), "")
		}
		a.renderView(a.stdout, items)
		return 0
	}

	it := &item{
		Hash:      newHash(s.hashTaken, a.rand),
		Text:      strings.Join(c.words, " "),
		State:     "open",
		CreatedAt: a.now().UTC().Truncate(time.Second),
		Cwd:       a.cwd,
	}
	probeGit(it)
	probeSession(it, a.env)
	if err := s.insert(it); err != nil {
		return a.fail(err)
	}
	return a.print(it, "%s  %s\n", it.Hash, it.Text)
}

// print writes v as JSON in --json mode, otherwise the formatted text.
func (a *app) print(v any, format string, args ...any) int {
	if a.json {
		if err := json.MarshalWrite(a.stdout, v); err != nil {
			return a.fail(err)
		}
		fmt.Fprintln(a.stdout)
		return 0
	}
	for i, v := range args {
		if s, ok := v.(string); ok {
			args[i] = clean(s)
		}
	}
	fmt.Fprintf(a.stdout, format, args...)
	return 0
}

func (a *app) fail(err error) int {
	code := 1
	if _, ok := err.(usageError); ok {
		code = 2
	}
	if a.json {
		json.MarshalWrite(a.stderr, map[string]string{"error": err.Error()})
		fmt.Fprintln(a.stderr)
		return code
	}
	fmt.Fprintf(a.stderr, "aaa: %s\n", clean(err.Error()))
	if code == 2 {
		fmt.Fprintln(a.stderr, usage)
	}
	return code
}
