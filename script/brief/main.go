// Command brief turns a voice's rules into the prompt you would otherwise
// have written by hand.
//
// The point is that it is derived. A skill carries its constraints twice --
// once as instructions the model reads and once as the judgment it is asked
// to apply -- and the two drift apart the moment a rule changes. Here the
// enforcement is the source and the instruction is the build artifact, so a
// brief that disagrees with the linter cannot survive a regenerate.
//
// Usage: go run ./script/brief -styles Voices/styles -out briefs
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// rule is the subset of a Vale rule a brief can say something about. Anything
// this struct does not model falls back to the rule's own message, which is
// the instruction the writer sees anyway.
type rule struct {
	Extends string `yaml:"extends"`
	Message string `yaml:"message"`
	Level   string `yaml:"level"`
	Scope   string `yaml:"scope"`
	// `sequence` also uses `tokens`, but as a list of maps, so this stays
	// untyped and the string case is filtered out in quoteAll.
	Tokens []any             `yaml:"tokens"`
	Swap   map[string]string `yaml:"swap"`
	Max    int               `yaml:"max"`
	Min    int               `yaml:"min"`
	Grade  float64           `yaml:"grade"`
	Match  string            `yaml:"match"`
	Dicts  []string          `yaml:"dictionaries"`
	Token  string            `yaml:"token"`
}

func main() {
	styles := flag.String("styles", "Voices/styles", "path to the styles directory")
	out := flag.String("out", "briefs", "directory to write briefs into")
	flag.Parse()

	names, err := voices(*styles)
	if err != nil {
		fail(err)
	}

	core, err := load(filepath.Join(*styles, "Voices"))
	if err != nil {
		fail(err)
	}

	for _, name := range names {
		rules, err := load(filepath.Join(*styles, name))
		if err != nil {
			fail(err)
		}
		body := render(name, core, rules)
		path := filepath.Join(*out, name+".md")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			fail(err)
		}
		fmt.Printf("%-10s %4d bytes  ~%d tokens\n", name, len(body), len(body)/4)
	}
}

// voices lists every style directory except the shared core and the config
// tree, which holds dictionaries rather than rules.
func voices(styles string) ([]string, error) {
	entries, err := os.ReadDir(styles)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "Voices" && e.Name() != "config" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

type named struct {
	name string
	rule rule
}

func load(dir string) ([]named, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.yml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	var out []named
	for _, path := range paths {
		data, rErr := os.ReadFile(path)
		if rErr != nil {
			return nil, rErr
		}
		var r rule
		if uErr := yaml.Unmarshal(data, &r); uErr != nil {
			return nil, fmt.Errorf("%s: %w", path, uErr)
		}
		out = append(out, named{strings.TrimSuffix(filepath.Base(path), ".yml"), r})
	}
	return out, nil
}

func render(voice string, core, rules []named) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# %s\n\n", voice)
	b.WriteString("Generated from the rules that enforce it; do not edit.\n")
	b.WriteString("`vale` checks all of this on every draft, so treat it as priming, not as a checklist to apply from memory.\n\n")

	fmt.Fprintf(&b, "## Always on\n\n")
	for _, n := range core {
		fmt.Fprintf(&b, "- %s\n", line(n))
	}
	fmt.Fprintf(&b, "\n## %s\n\n", voice)
	for _, n := range rules {
		fmt.Fprintf(&b, "- %s\n", line(n))
	}
	return b.String()
}

// line states one rule as an instruction. Each extension point carries the
// enumeration in a different key, so the phrasing follows the key rather than
// the message: `tokens` is a list of things not to write, `swap` is a list of
// replacements, `max` and `min` are budgets.
func line(n named) string {
	r := n.rule
	scope := r.Scope
	if scope == "" {
		scope = "block"
	}

	switch r.Extends {
	case "existence":
		if list := quoteAll(r.Tokens); list != "" {
			return fmt.Sprintf("**%s** — never write: %s", n.name, list)
		}
		// Every token was a pattern rather than a phrase, so there is no list
		// to hand over. The message is the instruction in that case: it is
		// what the writer sees when the rule fires.
		return fmt.Sprintf("**%s** — %s", n.name, plain(r.Message))
	case "substitution":
		pairs := make([]string, 0, len(r.Swap))
		for from, to := range r.Swap {
			pairs = append(pairs, fmt.Sprintf("%q → %q", from, to))
		}
		sort.Strings(pairs)
		return fmt.Sprintf("**%s** — replace: %s", n.name, strings.Join(pairs, "; "))
	case "occurrence":
		// `raw` is the whole file, which reads as "per document" rather than
		// as the name of a scope.
		if scope == "raw" {
			scope = "document"
		}
		if r.Min > 0 {
			return fmt.Sprintf("**%s** — at least %s per %s", n.name, countable(r.Token, r.Min), scope)
		}
		return fmt.Sprintf("**%s** — at most %s per %s", n.name, countable(r.Token, r.Max), scope)
	case "readability":
		return fmt.Sprintf("**%s** — reading grade at or below %g", n.name, r.Grade)
	case "capitalization":
		return fmt.Sprintf("**%s** — every %s in %s case", n.name, scope, strings.TrimPrefix(r.Match, "$"))
	case "spelling":
		return fmt.Sprintf("**%s** — only words in the %s dictionary", n.name, strings.Join(r.Dicts, ", "))
	default:
		return fmt.Sprintf("**%s** — %s", n.name, plain(r.Message))
	}
}

// countable turns an occurrence token back into something a reader can act
// on: `\b\w+\b` is "words", and an alternation is the list it enumerates.
func countable(token string, n int) string {
	switch token {
	case `\b\w+\b`, `\b[\w-]+\b`:
		return fmt.Sprintf("%d words", n)
	}
	inner := token
	inner = strings.TrimPrefix(inner, `\b`)
	inner = strings.TrimSuffix(inner, `\b`)
	inner = strings.TrimPrefix(inner, "(?:")
	inner = strings.TrimSuffix(inner, ")")
	if terms := alternates(inner); len(terms) > 1 {
		return fmt.Sprintf("%d of (%s)", n, strings.Join(terms, ", "))
	}
	if n == 1 {
		return fmt.Sprintf("one `%s`", token)
	}
	return fmt.Sprintf("%d of `%s`", n, token)
}

// quoteAll lists the tokens a reader could actually avoid writing. Patterns
// are skipped: "\\b(?:is|are) not (?:just )?[\\w\\s]{1,30}?" is not an
// instruction, and pretending otherwise is how a brief starts lying about
// what the rule does.
// alternates splits a top-level alternation into its members, ignoring the
// bars inside nested groups: "based(?!\\s+(?:on|upon))" is one term, not
// three. Lookarounds are dropped, since they disambiguate a term rather than
// name one.
func alternates(inner string) []string {
	var terms []string
	depth, start := 0, 0
	flush := func(end int) {
		term := inner[start:end]
		if i := strings.Index(term, "(?"); i >= 0 {
			term = term[:i]
		}
		if term = strings.TrimSpace(term); term != "" {
			terms = append(terms, term)
		}
	}
	for i, c := range inner {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		case '|':
			if depth == 0 {
				flush(i)
				start = i + 1
			}
		}
	}
	flush(len(inner))
	return terms
}

func quoteAll(tokens []any) string {
	var out []string
	for _, t := range tokens {
		s, ok := t.(string)
		if !ok || strings.ContainsAny(s, `\\[]{}^$|`) || strings.Contains(s, "(?") {
			continue
		}
		out = append(out, fmt.Sprintf("%q", s))
	}
	return strings.Join(out, ", ")
}

// plain strips the format verbs out of a rule message so it reads as a
// standing instruction rather than as a report about one match.
func plain(msg string) string {
	msg = strings.NewReplacer("'%s'. ", "", "'%s'", "it", "%s", "it", "%d", "n").Replace(msg)
	return strings.TrimSpace(msg)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "brief:", err)
	os.Exit(1)
}
