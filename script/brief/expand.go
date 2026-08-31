// Pattern machinery shared by the checks in main.go.
//
// A Vale token is a regex, and a brief is prose. These turn the first into the
// second where that is possible at all -- an alternation and an optional group
// become the phrases they match -- and refuse where it is not, so a character
// class or a quantifier is reported as a pattern rather than printed at a
// reader as though it were an instruction.
package main

import "strings"

// maxVariants guards against a combinatorial blowup: nested optional groups
// multiply, and a handful of them would expand into thousands of strings.
//
// It is deliberately far above any real list. A cap set near the length of an
// actual word list is worse than no cap at all, because growing the list past
// it drops the whole branch and turns the check off without saying so.
const maxVariants = 64

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
