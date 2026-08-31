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
		list, dropped := phrases(r.Tokens)
		if list != "" {
			return fmt.Sprintf("**%s** — never write: %s%s", n.name, list, note(dropped))
		}
		// Every token was a pattern rather than a phrase, so there is no list
		// to hand over. The message is the instruction in that case: it is
		// what the writer sees when the rule fires.
		return fmt.Sprintf("**%s** — %s", n.name, plain(r.Message))
	case "substitution":
		var pairs []string
		dropped := 0
		for from, to := range r.Swap {
			froms, ok := expand(from)
			if !ok {
				dropped++
				continue
			}
			for _, f := range froms {
				pairs = append(pairs, fmt.Sprintf("%q → %q", f, to))
			}
		}
		sort.Strings(pairs)
		if len(pairs) == 0 {
			return fmt.Sprintf("**%s** — %s", n.name, plain(r.Message))
		}
		return fmt.Sprintf("**%s** — replace: %s%s", n.name, strings.Join(pairs, "; "), note(dropped))
	case "occurrence":
		// `raw` is the whole file, which reads as "per document" rather than
		// as the name of a scope.
		if scope == "raw" {
			scope = "document"
		}
		bound, count := "at most", r.Max
		if r.Min > 0 {
			bound, count = "at least", r.Min
		}
		desc, dropped, ok := countable(r.Token, count)
		if !ok {
			return fmt.Sprintf("**%s** — %s", n.name, plain(r.Message))
		}
		return fmt.Sprintf("**%s** — %s %s per %s%s", n.name, bound, desc, scope, note(dropped))
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
// on: `\b\w+\b` is "words", and an alternation is the list it enumerates. It
// reports how many branches stayed patterns, and refuses rather than printing
// a regex, because a brief that says `(?m)^[*_>\s]{0,4}Why it matters:` has
// stopped being an instruction.
func countable(token string, n int) (string, int, bool) {
	switch token {
	case `\b\w+\b`, `\b[\w-]+\b`:
		return fmt.Sprintf("%d words", n), 0, true
	}
	terms, dropped := branches(token)
	if len(terms) == 0 {
		return "", 0, false
	}
	if len(terms) == 1 && dropped == 0 {
		if n == 1 {
			return fmt.Sprintf("one %q", terms[0]), 0, true
		}
		return fmt.Sprintf("%d of %q", n, terms[0]), 0, true
	}
	return fmt.Sprintf("%d of (%s)", n, strings.Join(terms, ", ")), dropped, true
}

// branches splits a token at its top-level alternation and expands each side.
// Anchors, word boundaries and lookarounds are stripped first: they say where a
// term may sit, not which term it is.
func branches(token string) ([]string, int) {
	var out []string
	dropped := 0
	for _, b := range split(token) {
		vs, ok := expand(strip(b))
		if !ok {
			dropped++
			continue
		}
		out = append(out, vs...)
	}
	return out, dropped
}

// split cuts a pattern at the `|` characters outside any group.
func split(pattern string) []string {
	rs := []rune(pattern)
	var out []string
	depth, start := 0, 0
	for i, c := range rs {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		case '|':
			if depth == 0 {
				out = append(out, string(rs[start:i]))
				start = i + 1
			}
		}
	}
	return append(out, string(rs[start:]))
}

// strip removes the position markers from a branch: the `(?m)` flag, the `^`
// and `$` anchors, `\b`, and any lookaround group with its contents.
func strip(branch string) string {
	for _, flag := range []string{"(?m)", "(?i)", "(?s)"} {
		branch = strings.ReplaceAll(branch, flag, "")
	}
	rs := []rune(branch)
	var b strings.Builder
	for i := 0; i < len(rs); {
		if rs[i] == '(' && i+2 < len(rs) && rs[i+1] == '?' &&
			(rs[i+2] == '!' || rs[i+2] == '=' || rs[i+2] == '<') {
			depth := 0
			for ; i < len(rs); i++ {
				if rs[i] == '(' {
					depth++
				} else if rs[i] == ')' {
					if depth--; depth == 0 {
						i++
						break
					}
				}
			}
			continue
		}
		if rs[i] == '\\' && i+1 < len(rs) && rs[i+1] == 'b' {
			i += 2
			continue
		}
		if rs[i] == '^' || rs[i] == '$' {
			i++
			continue
		}
		b.WriteRune(rs[i])
		i++
	}
	return b.String()
}

// quoteAll lists the tokens a reader could actually avoid writing. Patterns
// are skipped: "\\b(?:is|are) not (?:just )?[\\w\\s]{1,30}?" is not an
// instruction, and pretending otherwise is how a brief starts lying about
// what the rule does.
// phrases lists the tokens a reader could actually avoid writing, expanding
// the ones that are alternations or optional groups rather than dropping them.
// It also reports how many tokens it could not turn into a phrase, because a
// brief that quietly enumerates half a rule is how the instruction and the
// check drift apart again.
func phrases(tokens []any) (string, int) {
	var out []string
	dropped := 0
	for _, t := range tokens {
		// `sequence` also uses `tokens`, but as a list of maps.
		s, ok := t.(string)
		if !ok {
			dropped++
			continue
		}
		vs, ok := expand(s)
		if !ok {
			dropped++
			continue
		}
		for _, v := range vs {
			out = append(out, fmt.Sprintf("%q", v))
		}
	}
	return strings.Join(out, ", "), dropped
}

// note names what the list left out. Naming it is the point: the alternative
// is a brief that reads as complete while the linter checks more.
func note(dropped int) string {
	switch dropped {
	case 0:
		return ""
	case 1:
		return " (plus 1 pattern `vale` checks)"
	default:
		return fmt.Sprintf(" (plus %d patterns `vale` checks)", dropped)
	}
}

// maxVariants caps the expansion. A token that fans out further is a pattern
// in practice, whatever its syntax, and belongs in the dropped count.
const maxVariants = 12

// expand turns a token into the literal phrases it matches. It handles the
// two shapes these rules use -- an alternation inside `(?:...)` and a group
// made optional with `?` -- and refuses everything else, so a character class
// or a quantifier is reported as a pattern rather than printed as prose.
func expand(pattern string) ([]string, bool) {
	rs := []rune(pattern)
	vs, next, ok := parseAlt(rs, 0)
	if !ok || next != len(rs) {
		return nil, false
	}
	for _, v := range vs {
		if strings.TrimSpace(v) == "" {
			return nil, false
		}
	}
	return vs, len(vs) > 0
}

// parseAlt reads alternatives separated by a top-level `|`, stopping at the
// closing paren of the group it was called for.
func parseAlt(rs []rune, i int) ([]string, int, bool) {
	var all []string
	for {
		vs, next, ok := parseSeq(rs, i)
		if !ok {
			return nil, 0, false
		}
		all = append(all, vs...)
		if len(all) > maxVariants {
			return nil, 0, false
		}
		i = next
		if i < len(rs) && rs[i] == '|' {
			i++
			continue
		}
		return all, i, true
	}
}

// parseSeq reads a concatenation of literals and groups, crossing each group's
// alternatives into the variants built so far.
func parseSeq(rs []rune, i int) ([]string, int, bool) {
	cur := []string{""}
	for i < len(rs) {
		c := rs[i]
		if c == '|' || c == ')' {
			return cur, i, true
		}
		if c == '(' {
			if !hasPrefix(rs, i, "(?:") {
				return nil, 0, false
			}
			vs, next, ok := parseAlt(rs, i+3)
			if !ok || next >= len(rs) || rs[next] != ')' {
				return nil, 0, false
			}
			next++
			if next < len(rs) && rs[next] == '?' {
				vs = append(vs, "")
				next++
			}
			cur = cross(cur, vs)
			i = next
		} else {
			if strings.ContainsRune(`\[]{}^$.*+?`, c) {
				return nil, 0, false
			}
			for j := range cur {
				cur[j] += string(c)
			}
			i++
		}
		if len(cur) > maxVariants {
			return nil, 0, false
		}
	}
	return cur, i, true
}

func hasPrefix(rs []rune, i int, s string) bool {
	return i+len([]rune(s)) <= len(rs) && string(rs[i:i+len([]rune(s))]) == s
}

func cross(prefixes, suffixes []string) []string {
	out := make([]string, 0, len(prefixes)*len(suffixes))
	for _, p := range prefixes {
		for _, s := range suffixes {
			out = append(out, p+s)
		}
	}
	return out
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
