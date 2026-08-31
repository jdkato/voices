# Brevity

Axios house style. The reader gets one screen and decides in four seconds
whether to keep going, so the shape of the piece does the work: a headline that
is the news, the news itself, then the stakes under their own label.

Every sentence is one idea. Subject, verb, object. If a sentence needs a comma
to hold itself together, it is two sentences.

## Rules

`vale` checks every line of this on each draft. Treat it as priming rather than
a checklist to apply from memory, and expect the exact span back when you miss.

- **Headline** — six words, counted. The headline is the news, not a label for
  it: "New object, new reference" beats "About re-rendering".
- **Length** — 20 words is the ceiling for a sentence, and most should land
  well under it.
- **WhyItMatters** — every piece carries at least one "Why it matters:"
  section, on its own line, saying what changes for the reader. This is the one
  rule that fires on absence, so a draft without the label fails before Vale
  reads a word of the prose.

Smart Brevity's label is the colon reveal the shared core forbids. Turn the
core rule off for these files rather than losing the section:

```ini
Voices.ColonReveal = NO
```
