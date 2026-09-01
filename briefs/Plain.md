# Plain

Write so a smart reader outside your field gets it on the first pass. That
means short words where short words exist, and a named actor in front of every
verb.

Keep the substance. Plain English strips the packaging, not the content: the
numbers, the mechanism, and the caveat all survive. Only the fog goes.

## Rules

`vale` checks every line of this on each draft. Treat it as priming rather than
a checklist to apply from memory, and expect the exact span back when you miss.

- **Length** — 35 words is the ceiling for a sentence. A long sentence usually
  hides a list; write the list.
- **Nominalization** — use the verb, not the noun built from it. Each of these
  carries its replacement, so an agent applies it without deciding anything:
  provide support for → support, provides support for → supports, provided
  support for → supported; perform an analysis → analyze, performs an analysis
  → analyzes, performed an analysis → analyzed; give consideration to →
  consider, gives consideration to → considers, given consideration to →
  considered; conduct an investigation → investigate, conducts an investigation
  → investigates, conducted an investigation → investigated; reach a conclusion
  → conclude, reaches a conclusion → concludes, reached a conclusion →
  concluded.
- **Passive** — name who acts. "The daemon retries the write" beats "the write
  is retried". Vale flags every passive, including the ones that name their
  actor, so if you keep one, keep it on purpose. Marking it in the file says so
  out loud: `<!-- vale Plain.Passive = NO -->`.
- **Readability** — grade 12 or below, by Flesch-Kincaid and Gunning Fog. Both
  read sentence length and syllables, so shorter sentences and shorter words
  are the whole lever.
- **Wordiness** — the wordy phrase has a short form: "due to the fact that" is
  "because", "in the event that" is "if", "prior to" is "before". There are 75
  of these and the brief does not list them, because each alert arrives with
  its own replacement. Write plainly and you will not meet them.
