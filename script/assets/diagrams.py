#!/usr/bin/env python3
"""Emit the README's diagrams as SVG files.

GitHub strips inline <svg> from Markdown, so these ship as files and are
referenced through <picture>. An <img> renders in its own document, which also
means currentColor never reaches it -- so every diagram is emitted twice, once
per theme, from a single definition here.

Usage: python3 script/assets/diagrams.py
"""

import pathlib

OUT = pathlib.Path(__file__).resolve().parents[2] / ".github/assets"

THEMES = {
    "light": dict(fg="#18181b", muted="#71717a", line="#a1a1aa", border="#d4d4d8",
                  card="#ffffff", lime="#65a30d", limeline="#84cc16",
                  limefill="#f7fee7", flag="#dc2626"),
    "dark": dict(fg="#e4e4e7", muted="#a1a1aa", line="#71717a", border="#3f3f46",
                 card="#18181b", lime="#a3e635", limeline="#65a30d",
                 limefill="#1a2e05", flag="#f87171"),
}

MONO = "ui-monospace,SFMono-Regular,Menlo,Consolas,monospace"
SANS = "-apple-system,BlinkMacSystemFont,Segoe UI,Helvetica,Arial,sans-serif"

# Mono advance width at 13px. Monospace metrics vary by platform, and a
# renderer that picks a wider face would push the text out of its panel and
# slide the underlines off the words they mark -- so every line of the sample
# prose carries an explicit textLength and is fitted to this grid.
CH = 7.82


def esc(s):
    return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def box(x, y, w, h, label, t, accent=False):
    fill = t["limefill"] if accent else t["card"]
    stroke = t["limeline"] if accent else t["border"]
    color = t["lime"] if accent else t["fg"]
    return (f'<rect x="{x}" y="{y}" width="{w}" height="{h}" rx="8" fill="{fill}" '
            f'stroke="{stroke}" stroke-width="1"/>'
            f'<text x="{x + w / 2}" y="{y + h / 2 + 4.5}" text-anchor="middle" '
            f'font-family="{MONO}" font-size="13" fill="{color}">{esc(label)}</text>')


def squiggle(x, y, width, color):
    """A wavy underline, the way an editor marks a span rather than a line."""
    d, cursor, up = [f"M{x} {y}"], x, True
    while cursor < x + width:
        step = min(4, x + width - cursor)
        d.append(f"q {step / 2} {-3 if up else 3} {step} 0")
        cursor += step
        up = not up
    return (f'<path d="{" ".join(d)}" fill="none" stroke="{color}" '
            f'stroke-width="1.4" opacity="0.9"/>')


def hero(t):
    """The product in one image: a draft, what the rules caught, the rewrite."""
    s = [f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 320" '
         f'width="1000" height="320">',
         f'<defs><marker id="h" markerWidth="9" markerHeight="9" refX="8" refY="4" '
         f'orient="auto"><path d="M0 0 L9 4 L0 8 z" fill="{t["limeline"]}"/></marker></defs>']

    s.append(f'<text x="0" y="30" font-family="{MONO}" font-size="26" font-weight="600" '
             f'fill="{t["fg"]}">Voices</text>')
    s.append(f'<text x="0" y="54" font-family="{SANS}" font-size="14" '
             f'fill="{t["muted"]}">The output-style catalog, as a linter.</text>')

    draft = [
        ("Here's the thing: it's worth noting", [(0, 16), (18, 17)]),
        ("that the reason your component is", [(5, 27)]),
        ("re-rendering is likely because you", [(19, 15)]),
        ("may want to consider memoization.", [(0, 20)]),
    ]
    s.append(f'<rect x="0" y="86" width="410" height="200" rx="10" fill="{t["card"]}" '
             f'stroke="{t["border"]}" stroke-width="1"/>')
    s.append(f'<text x="20" y="112" font-family="{MONO}" font-size="11" '
             f'fill="{t["muted"]}">draft</text>')
    s.append(f'<text x="390" y="112" text-anchor="end" font-family="{MONO}" '
             f'font-size="11" fill="{t["flag"]}">17 alerts &#183; exit 1</text>')
    for i, (line, spans) in enumerate(draft):
        y = 145 + i * 30
        s.append(f'<text x="20" y="{y}" font-family="{MONO}" font-size="13" '
                 f'textLength="{len(line) * CH:.1f}" lengthAdjust="spacingAndGlyphs" '
                 f'fill="{t["fg"]}">{esc(line)}</text>')
        for start, length in spans:
            s.append(squiggle(20 + start * CH, y + 6, length * CH, t["flag"]))

    s.append(f'<line x1="440" y1="186" x2="530" y2="186" stroke="{t["limeline"]}" '
             f'stroke-width="1.5" marker-end="url(#h)"/>')
    s.append(f'<text x="485" y="176" text-anchor="middle" font-family="{MONO}" '
             f'font-size="11" fill="{t["muted"]}">vale</text>')

    rewrite = [
        "Your component re-renders because",
        "you create a new object on every",
        "render. React compares props by",
        "reference, so the memo check fails.",
    ]
    s.append(f'<rect x="560" y="86" width="440" height="200" rx="10" '
             f'fill="{t["card"]}" stroke="{t["limeline"]}" stroke-width="1"/>')
    s.append(f'<text x="580" y="112" font-family="{MONO}" font-size="11" '
             f'fill="{t["muted"]}">rewrite</text>')
    s.append(f'<text x="980" y="112" text-anchor="end" font-family="{MONO}" '
             f'font-size="11" fill="{t["lime"]}">clean &#183; exit 0</text>')
    for i, line in enumerate(rewrite):
        s.append(f'<text x="580" y="{145 + i * 30}" font-family="{MONO}" font-size="13" '
                 f'textLength="{len(line) * CH:.1f}" lengthAdjust="spacingAndGlyphs" '
                 f'fill="{t["fg"]}">{esc(line)}</text>')

    voices = ["Direct", "Plain", "Unslop", "Brevity", "Simple", "GenZ"]
    x = 0
    for name in voices:
        s.append(f'<text x="{x}" y="312" font-family="{MONO}" font-size="12" '
                 f'fill="{t["muted"]}">{name}</text>')
        x += len(name) * 7.4 + 24
    s.append('</svg>')
    return "\n".join(s)


def loop(t):
    s = [f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 130" width="640" height="130">',
         f'<defs><marker id="a" markerWidth="8" markerHeight="8" refX="7" refY="3.5" orient="auto">'
         f'<path d="M0 0 L8 3.5 L0 7 z" fill="{t["line"]}"/></marker>'
         f'<marker id="b" markerWidth="8" markerHeight="8" refX="7" refY="3.5" orient="auto">'
         f'<path d="M0 0 L8 3.5 L0 7 z" fill="{t["limeline"]}"/></marker></defs>']
    for x, label in [(8, "agent writes"), (176, "vale --ext=.md"), (344, "agent fixes")]:
        s.append(box(x, 34, 144, 44, label, t))
    for x in (152, 320):
        s.append(f'<line x1="{x}" y1="56" x2="{x + 16}" y2="56" stroke="{t["line"]}" '
                 f'stroke-width="1.5" marker-end="url(#a)"/>')
    s.append(f'<line x1="496" y1="56" x2="512" y2="56" stroke="{t["limeline"]}" '
             f'stroke-width="1.5" marker-end="url(#b)"/>')
    s.append(box(520, 34, 112, 44, "exit 0", t, accent=True))
    # The failing branch runs back underneath: the exit code is what puts the
    # agent into the loop again without being asked.
    s.append(f'<path d="M416 82 L416 108 L248 108 L248 82" fill="none" stroke="{t["line"]}" '
             f'stroke-width="1.5" stroke-dasharray="4 4" marker-end="url(#a)"/>')
    s.append(f'<text x="332" y="123" text-anchor="middle" font-family="{MONO}" '
             f'font-size="11" fill="{t["muted"]}">exit 1 &#183; alerts</text>')
    s.append("</svg>")
    return "\n".join(s)


def split(t):
    s = [f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 640 210" width="640" height="210">',
         f'<defs><marker id="a" markerWidth="8" markerHeight="8" refX="7" refY="3.5" orient="auto">'
         f'<path d="M0 0 L8 3.5 L0 7 z" fill="{t["line"]}"/></marker>'
         f'<marker id="b" markerWidth="8" markerHeight="8" refX="7" refY="3.5" orient="auto">'
         f'<path d="M0 0 L8 3.5 L0 7 z" fill="{t["limeline"]}"/></marker></defs>']
    s.append(f'<rect x="8" y="72" width="150" height="62" rx="8" fill="{t["card"]}" '
             f'stroke="{t["border"]}" stroke-width="1"/>')
    s.append(f'<text x="83" y="99" text-anchor="middle" font-family="{MONO}" font-size="13" '
             f'fill="{t["fg"]}">gen-z.md</text>')
    s.append(f'<text x="83" y="117" text-anchor="middle" font-family="{SANS}" font-size="11" '
             f'fill="{t["muted"]}">745 tokens</text>')
    for y, label, note, accent in [
        (14, "persona", "stays a prompt", False),
        (84, "constraints", "become rules", True),
        (154, "guardrails", "free: Vale never sees code", False),
    ]:
        stroke = t["limeline"] if accent else t["line"]
        s.append(f'<path d="M162 103 C 208 103, 208 {y + 22}, 254 {y + 22}" fill="none" '
                 f'stroke="{stroke}" stroke-width="1.5" '
                 f'marker-end="url(#{"b" if accent else "a"})"/>')
        s.append(box(262, y, 152, 44, label, t, accent))
        s.append(f'<text x="428" y="{y + 27}" font-family="{SANS}" font-size="12" '
                 f'fill="{t["muted"]}">{note}</text>')
    s.append("</svg>")
    return "\n".join(s)


def main():
    OUT.mkdir(parents=True, exist_ok=True)
    for name, fn in (("hero", hero), ("loop", loop), ("split", split)):
        for theme, palette in THEMES.items():
            path = OUT / f"{name}-{theme}.svg"
            path.write_text(fn(palette) + "\n")
            print(f"{path.name:20} {path.stat().st_size:>6,} bytes")


if __name__ == "__main__":
    main()
