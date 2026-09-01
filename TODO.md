# TODO

Open work, roughly in the order it costs someone something. Each item says how
it was measured, because two of these were invisible until something forced
them to show.

## Blocking anyone who follows the README

- [ ] **Cut a release.** The quickstart tells people to write
  `Packages = https://github.com/jdkato/voices/releases/latest/download/Voices.zip`
  and that URL returns 404. There are no tags, and the workflow only
  uploads the archive on `refs/tags/*`, so `vale sync` fails for every new
  user. Tag `v0.1.0` and the existing job does the rest.

## Coverage of the rules themselves

- [ ] **Take the coverage check down to the token.** `./test.sh` proves every
  rule fires somewhere. It says nothing about the entries inside a rule,
  and the gap is wide: 16 of 170 list entries appear in any golden, about
  9%. `Plain.Wordiness` exercises 1 of its 75 pairs, `Voices.Banned` 2 of
  16, `Plain.Nominalization` 1 of 15. Fourteen banned words could carry a
  typo today and nothing would notice.

  The machinery already exists. `script/brief` expands a token into the
  phrases it matches; point the same expander at the goldens instead of at
  the briefs. Regex tokens stay uncheckable and should be counted and
  reported, the way the brief check reports what it takes on trust.

- [ ] **Decide what token coverage should cost.** Exercising 170 entries means
  fixtures far larger than the six-line draft, and the goldens are read by
  people. One dense file per rule, checked for alert count rather than
  exact text, may be the better trade.

## Vale itself

None of these are fixable from a style package. They need
[vale-cli/vale](https://github.com/vale-cli/vale), which was out of scope for
the session that wrote them down.

- [ ] **`vale --unused`.** A rule that matches nothing loads, runs, and reports
  success. Every package author either builds the workaround in `test.sh`
  or ships dead rules unknowingly. Four dead rules were found here that
  way, and the fourth survived a test suite built to catch exactly that.

- [ ] **Style composition.** Vale has no include, so the GenZ slang token lives
  in three files and `test.sh` fails if the copies drift. Painful at 29
  rules and disqualifying at 200.

- [ ] **Position for `min` occurrence rules.** A shortfall reports once per
  file at 1:1 rather than at the block that came up short. A person infers
  it. A model needs the span, which is the reason an alert beats a prompt
  line.

- [ ] **Inline exemptions fail silently on a deep indent.** A
  `<!-- vale Rule = NO -->` inside a list item works at a two-space
  continuation and does nothing at six, while the prose around it is still
  checked. The writer sees the alert persist and has no way to tell why. The
  directive should either apply or report that it was ignored. This file uses
  two-space continuations because of it.

- [ ] **Let a `sequence` token see its neighbours.** `sequence` reads the
  tagger and knows a participle; its patterns match one token at a time, so
  it cannot spare "was fixed by Dana". `existence` sees the span but has no
  tagger, so it reads "the light is red" as a passive. `Plain.Passive`
  picks the first and documents the trade. Either shape alone leaves the
  agentless rule out of reach.

## Rules

- [ ] **Grow `Voices.InflatedWords`.** Nine words carry a mechanical fix and 16
  stay in `Banned` for judgment. Some of the 16 have one right answer in
  practice and could move. Each move has to survive being applied without
  thought, so this is slow work, one word at a time.

- [ ] **Deletion fixes.** `Direct.Hedging` and `Voices.Recap` are detect-only.
  `action: remove` works, but the cut leaves wreckage.
  <!-- vale Direct.Hedging = NO -->
  Removing "it's worth noting" from "it's worth noting that it helps"
  strands the "that", and nothing recapitalizes after "In conclusion" goes.
  <!-- vale Direct.Hedging = YES -->
  Both need an action that rewrites rather than excises.

- [ ] **Guard `Unslop.Vague`.** "things" and "several" fire on ordinary prose.
  The GenZ token shows the shape. A lookbehind or lookahead keeps the
  slang sense and drops the plain one.

- [ ] **`Brevity.WhyItMatters` still reads raw text.** Anchoring to the start of
  a line stops a fenced code block from satisfying it in the common case. A
  code block whose own line starts with the label still would.

- [ ] **`GenZ` W and L cannot be stated.** The modifier branch expands to 26
  variants, past the cap. So the brief describes it in prose and the check
  records it as taken on trust. Either shape is fine, as long as the count
  stays visible.

## Housekeeping

- [ ] `script/tokens/count.py` needs `tiktoken` and nothing declares it. A
  `requirements.txt` beside it, or a note in its docstring.
- [ ] The measured prompt is a copy of no-ai-slop's `SKILL.md` pinned at
  `b53e265`. Worth refreshing, and worth saying in the README that the
  figure is a snapshot.
- [ ] Delete `claude/thoughts-2s8gpi`. It is merged and identical to `main`.
