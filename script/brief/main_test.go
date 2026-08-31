package main

import "testing"

// A checker that cannot fail is worth exactly as much as a Vale rule that
// matches nothing, and fails the same silent way. These are the drifts it
// exists to catch, each stated as a brief that should not pass.
func TestCheckCatchesDrift(t *testing.T) {
	banned := named{"Banned", rule{
		Extends: "existence",
		Tokens:  []any{"delve", "leverage", "unlock"},
	}}
	length := named{"Length", rule{
		Extends: "occurrence",
		Token:   `\b\w+\b`,
		Max:     25,
	}}
	slang := named{"Density", rule{
		Extends: "occurrence",
		Token:   `\b(?:cooked|rizz|mid|based)\b`,
		Max:     1,
	}}

	cases := []struct {
		name  string
		brief string
		rules []named
		gaps  int
	}{{
		name:  "a stated rule with every token named passes",
		brief: "- **Banned** — never write delve, leverage, or unlock.",
		rules: []named{banned},
		gaps:  0,
	}, {
		name:  "a token the brief never names is a gap",
		brief: "- **Banned** — never write delve or leverage.",
		rules: []named{banned},
		gaps:  1,
	}, {
		name:  "a rule with no line of its own is a gap",
		brief: "Avoid delve, leverage, and unlock.",
		rules: []named{banned},
		gaps:  1,
	}, {
		name:  "a line naming no rule is a gap, so an invented rule shows up",
		brief: "- **Banned** — delve, leverage, unlock.\n- **Cadence** — vary it.",
		rules: []named{banned},
		gaps:  1,
	}, {
		name:  "a bound the brief disagrees with is a gap",
		brief: "- **Length** — 30 words is the ceiling.",
		rules: []named{length},
		gaps:  1,
	}, {
		name:  "a bound written as a word counts as stated",
		brief: "- **Length** — twenty-five words is the ceiling.",
		rules: []named{length},
		gaps:  0,
	}, {
		name:  "every term of a vocabulary has to be named, not just one",
		brief: "- **Density** — at most 1 of cooked, rizz, mid.",
		rules: []named{slang},
		gaps:  1,
	}, {
		name:  "the contraction and the long form are the same instruction",
		brief: "- **Hedging** — never write \"it's worth noting\".",
		rules: []named{{"Hedging", rule{
			Extends: "existence",
			Tokens:  []any{"it(?:'s| is) worth noting"},
		}}},
		gaps: 0,
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gaps, _ := check(c.brief, c.rules)
			if len(gaps) != c.gaps {
				t.Errorf("got %d gaps, want %d: %v", len(gaps), c.gaps, gaps)
			}
		})
	}
}

// A cap set near the length of a real word list silently drops the branch and
// stops checking it, which is how a growing vocabulary turns the check off.
// This list is longer than any cap should be.
func TestLongVocabularyStaysChecked(t *testing.T) {
	long := named{"Density", rule{
		Extends: "occurrence",
		Max:     1,
		Token: `\b(?:cooked|rizz|mid|based|aura|delulu|skibidi|bussin|` +
			`goated|lowkey|highkey|glazing|sigma|rent free)\b`,
	}}
	brief := "- **Density** — at most 1 of cooked, rizz, mid, based, aura, " +
		"delulu, skibidi, bussin, goated, lowkey, highkey, glazing, sigma."

	gaps, _ := check(brief, []named{long})
	if len(gaps) != 1 {
		t.Fatalf("got %d gaps, want 1 (the unnamed term): %v", len(gaps), gaps)
	}
}

// The expander is the whole basis of the token checks, so its refusals matter
// as much as its expansions: anything it renders wrong would be checked wrong.
func TestExpand(t *testing.T) {
	cases := []struct {
		pattern string
		want    []string
		ok      bool
	}{
		{"delve", []string{"delve"}, true},
		{"it(?:'s| is) worth noting",
			[]string{"it's worth noting", "it is worth noting"}, true},
		{"(?:you )?may want to", []string{"you may want to", "may want to"}, true},
		{`[\w\s]{0,20}`, nil, false},
		{`provide(?:s|d)? support for`, []string{
			"provides support for", "provided support for", "provide support for",
		}, true},
		{`\bmid\b`, nil, false}, // escapes are for branches() to strip first
	}
	for _, c := range cases {
		got, ok := expand(c.pattern)
		if ok != c.ok {
			t.Errorf("expand(%q) ok = %v, want %v", c.pattern, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("expand(%q) = %v, want %v", c.pattern, got, c.want)
			continue
		}
		seen := map[string]bool{}
		for _, g := range got {
			seen[g] = true
		}
		for _, w := range c.want {
			if !seen[w] {
				t.Errorf("expand(%q) = %v, missing %q", c.pattern, got, w)
			}
		}
	}
}

// branches() is what lets a guarded token still be read as a word list.
func TestBranchesStripsGuards(t *testing.T) {
	token := `\b(?:cooked|rizz)\b|(?<!-)\bmid\b(?!-)|\b(?:the|massive)\s+[WL]\b`
	terms, dropped := branches(token)

	want := map[string]bool{"cooked": true, "rizz": true, "mid": true}
	if len(terms) != len(want) {
		t.Fatalf("got %v, want the three literal terms", terms)
	}
	for _, term := range terms {
		if !want[term] {
			t.Errorf("unexpected term %q", term)
		}
	}
	if dropped != 1 {
		t.Errorf("got %d dropped branches, want 1 (the W/L pattern)", dropped)
	}
}
