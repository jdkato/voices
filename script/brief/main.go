// Command brief checks that each hand-written brief still states what the
// rules enforce.
//
// The briefs used to be generated from the rules, on the theory that a derived
// instruction cannot drift from the check. What that actually bought was
// reproducibility, not truth: the renderer dropped three of Direct's five
// hedges and printed `ha(?:s|ve|d) the ability to` at the reader as prose, and
// `git diff --exit-code briefs/` stayed green throughout, because the output
// was still a function of the rules. It was wrong and consistent.
//
// So the direction is reversed. A person writes the brief -- including the half
// no renderer reaches, which is how the voice should sound -- and this walks
// the rules to find what went unsaid:
//
//   - Every rule is named in a `- **Name** —` line, and every such line names a
//     rule that exists. A rule cannot be added, dropped, or renamed in silence.
//   - Every token list is represented. A word list has to appear in the brief
//     word for word; for a token with several forms, any one of them will do.
//   - Every bound appears, as a numeral or as its English word.
//
// What this cannot check is a true-sounding sentence about a rule that is not
// there. The bold-name scan catches the usual shape of that. The rest of the
// prose is on whoever wrote it.
//
// Usage: go run ./script/brief -styles Voices/styles -briefs briefs
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// core is the shared style, and the brief that carries it. Every voice turns
// it on, so it is stated once and pasted alongside whichever voice you pick.
const (
	core      = "Voices"
	coreBrief = "Core"
)

// rule is the subset of a Vale rule a brief can be checked against. Anything
// this struct does not model contributes its name and nothing else.
type rule struct {
	Extends string `yaml:"extends"`
	Message string `yaml:"message"`
	Level   string `yaml:"level"`
	Scope   string `yaml:"scope"`
	// `sequence` also uses `tokens`, but as a list of maps, so this stays
	// untyped and the string case is filtered out where it is read.
	Tokens []any             `yaml:"tokens"`
	Swap   map[string]string `yaml:"swap"`
	Max    int               `yaml:"max"`
	Min    int               `yaml:"min"`
	Grade  float64           `yaml:"grade"`
	Match  string            `yaml:"match"`
	Dicts  []string          `yaml:"dictionaries"`
	Token  string            `yaml:"token"`
}

type named struct {
	name string
	rule rule
}

func main() {
	styles := flag.String("styles", "Voices/styles", "path to the styles directory")
	briefs := flag.String("briefs", "briefs", "directory holding the briefs")
	flag.Parse()

	names, err := voices(*styles)
	if err != nil {
		fail(err)
	}

	// The shared style and its brief are named differently; every voice
	// answers to itself.
	pairs := []struct{ style, brief string }{{core, coreBrief}}
	for _, n := range names {
		pairs = append(pairs, struct{ style, brief string }{n, n})
	}

	total := 0
	for _, p := range pairs {
		style, brief := p.style, p.brief
		rules, lErr := load(filepath.Join(*styles, style))
		if lErr != nil {
			fail(lErr)
		}
		path := filepath.Join(*briefs, brief+".md")
		text, rErr := os.ReadFile(path)
		if rErr != nil {
			fail(rErr)
		}

		gaps, unchecked := check(string(text), rules)
		if len(gaps) == 0 {
			// Patterns carry no phrase to look for, so the brief's account
			// of them rests on whoever wrote it. Saying how many keeps that
			// visible instead of letting it read as full coverage.
			note := ""
			if unchecked > 0 {
				note = fmt.Sprintf(", %d pattern(s) taken on trust", unchecked)
			}
			fmt.Printf("ok   %-16s %d rules stated%s\n", path, len(rules), note)
			continue
		}
		total += len(gaps)
		fmt.Printf("FAIL %s\n", path)
		for _, g := range gaps {
			fmt.Printf("       %s\n", g)
		}
	}

	if total > 0 {
		fmt.Fprintf(os.Stderr, "\n%d gap(s) between the briefs and the rules\n", total)
		os.Exit(1)
	}
}

// listed matches the one line shape a brief uses to state a rule. Holding the
// rules to a fixed shape is what lets the check run in both directions.
var listed = regexp.MustCompile(`(?m)^- \*\*([A-Za-z]+)\*\*`)

// check reports every way the brief and the rules disagree, and counts the
// patterns it had no phrase to check.
func check(text string, rules []named) ([]string, int) {
	var gaps []string
	unchecked := 0
	body := norm(text)

	stated := map[string]bool{}
	for _, m := range listed.FindAllStringSubmatch(text, -1) {
		stated[m[1]] = true
	}
	known := map[string]bool{}
	for _, n := range rules {
		known[n.name] = true
	}
	for name := range stated {
		if !known[name] {
			gaps = append(gaps, fmt.Sprintf("**%s** names no rule here", name))
		}
	}

	for _, n := range rules {
		if !stated[n.name] {
			gaps = append(gaps, fmt.Sprintf("%s: no `- **%s** —` line", n.name, n.name))
			continue
		}
		wants, opaque := requires(n.rule)
		unchecked += opaque
		for _, want := range wants {
			if !want.met(body) {
				gaps = append(gaps, fmt.Sprintf("%s: %s", n.name, want))
			}
		}
	}
	sort.Strings(gaps)
	return gaps, unchecked
}

// want is one thing the brief has to say. Several alternatives mean the brief
// may pick whichever form reads best: "it's worth noting" and "it is worth
// noting" are the same instruction.
type want struct {
	kind        string
	alternative []string
}

func (w want) met(body string) bool {
	for _, a := range w.alternative {
		if a != "" && strings.Contains(body, norm(a)) {
			return true
		}
	}
	return false
}

func (w want) String() string {
	if len(w.alternative) == 1 {
		return fmt.Sprintf("%s %q goes unstated", w.kind, w.alternative[0])
	}
	return fmt.Sprintf("%s %q goes unstated (or any of %d forms)",
		w.kind, w.alternative[0], len(w.alternative))
}

// requires lists what a brief has to contain for one rule. A word list has to
// be there word for word, because a brief that names four of five banned words
// is how the fifth gets written. A pattern contributes nothing beyond its
// rule's name, which is checked separately.
func requires(r rule) ([]want, int) {
	var out []want
	opaque := 0
	switch r.Extends {
	case "existence":
		// One requirement per token, met by any form of it. These rules
		// keep a word per token, so a list is still checked in full: drop
		// a word from `Banned` and its line stops being satisfied.
		for _, t := range r.Tokens {
			s, ok := t.(string)
			if !ok {
				continue
			}
			terms, dropped := branches(s)
			opaque += dropped
			if len(terms) > 0 {
				out = append(out, want{"token", terms})
			}
		}
	case "substitution":
		keys := make([]string, 0, len(r.Swap))
		for from := range r.Swap {
			keys = append(keys, from)
		}
		sort.Strings(keys)
		for _, from := range keys {
			terms, dropped := branches(from)
			opaque += dropped
			if len(terms) > 0 {
				out = append(out, want{"replacement of", terms})
			}
		}
	case "occurrence":
		// The token of an occurrence rule is a vocabulary rather than a
		// single phrase, so every term in it has to be named. A brief that
		// sets a slang budget without saying which words are slang has
		// stated the arithmetic and left out the voice.
		terms, dropped := branches(r.Token)
		opaque += dropped
		for _, t := range terms {
			out = append(out, want{"term", []string{t}})
		}
		if r.Min > 0 {
			out = append(out, want{"bound", counts(r.Min)})
		} else {
			out = append(out, want{"bound", counts(r.Max)})
		}
	case "readability":
		out = append(out, want{"grade", counts(int(r.Grade))})
	case "spelling":
		for _, d := range r.Dicts {
			out = append(out, want{"dictionary", []string{d}})
		}
	}
	return out, opaque
}

// counts gives the forms a bound may take in prose. "Six is the budget" and
// "at most 6" are the same instruction, and a brief should be free to read
// like a sentence.
func counts(n int) []string {
	words := []string{
		"zero", "one", "two", "three", "four", "five", "six", "seven",
		"eight", "nine", "ten", "eleven", "twelve", "thirteen", "fourteen",
		"fifteen", "sixteen", "seventeen", "eighteen", "nineteen", "twenty",
	}
	out := []string{fmt.Sprint(n)}
	if n >= 0 && n < len(words) {
		out = append(out, words[n])
	}
	if n == 25 {
		out = append(out, "twenty-five")
	}
	if n == 35 {
		out = append(out, "thirty-five")
	}
	return out
}

// norm folds the differences that do not change what a brief says: case, the
// two apostrophes, and the contraction of "is". Without the last one a brief
// has to spell out both halves of every `(?:'s| is)` token to satisfy a check
// that neither form would fail a reader.
func norm(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "'s ", " is ")
	return strings.Join(strings.Fields(s), " ")
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
		if e.IsDir() && e.Name() != core && e.Name() != "config" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
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

func fail(err error) {
	fmt.Fprintln(os.Stderr, "brief:", err)
	os.Exit(1)
}
