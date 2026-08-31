#!/usr/bin/env python3
"""Count the tokens behind the figures in the README.

Two backends, because they answer different questions:

  anthropic  Claude's own tokenizer, via the count_tokens endpoint. This is
             the authoritative number for anyone running these rules against
             Claude, and it names the model it counted for. Needs
             ANTHROPIC_API_KEY.

  tiktoken   OpenAI's o200k_base, offline. Anthropic publishes no offline
             tokenizer for Claude 3 and later, so this is the reproducible
             stand-in: it is a real BPE count rather than an estimate, and it
             lands within a few percent on English prose.

Usage:
    python3 script/tokens/count.py                     # tiktoken
    python3 script/tokens/count.py --backend anthropic # Claude, needs a key
    python3 script/tokens/count.py --model claude-opus-5
"""

import argparse
import json
import os
import pathlib
import subprocess
import sys
import tempfile

ROOT = pathlib.Path(__file__).resolve().parents[2]
DEFAULT_MODEL = "claude-opus-5"

# The prompt this package is measured against. Cached locally so a rate limit
# or an upstream edit cannot silently move the number.
SKILL_REPO = "https://github.com/petergyang/no-ai-slop"
SKILL_PATH = "skills/no-ai-slop/SKILL.md"
SKILL_SHA = "b53e2659b986093f7c681d8b4e998715e90da2a2"  # 2026-08-06
SKILL_URL = f"https://raw.githubusercontent.com/petergyang/no-ai-slop/{SKILL_SHA}/{SKILL_PATH}"


def count_tiktoken(texts):
    import tiktoken

    enc = tiktoken.get_encoding("o200k_base")
    return {k: len(enc.encode(v)) for k, v in texts.items()}, "o200k_base (tiktoken)"


def count_anthropic(texts, model):
    import anthropic

    client = anthropic.Anthropic()
    out = {}
    for k, v in texts.items():
        if not v:
            out[k] = 0
            continue
        # count_tokens charges the request envelope too, so an empty body is
        # measured once and subtracted -- what we want is the cost of the
        # content, not of asking.
        r = client.messages.count_tokens(
            model=model, messages=[{"role": "user", "content": v}]
        )
        out[k] = r.input_tokens
    base = client.messages.count_tokens(
        model=model, messages=[{"role": "user", "content": "."}]
    ).input_tokens - 1
    return {k: max(v - base, 0) for k, v in out.items()}, f"{model} (count_tokens)"


def vale(voice, path):
    """The alerts a draft produces under one voice, as an agent would read them."""
    # The length rules extend Std, so Std has to be on the StylesPath --
    # present, not enabled, the same arrangement test.sh uses.
    std = pathlib.Path(os.environ.get("STD", ROOT.parent / "Std" / "Std"))
    if not std.is_dir():
        sys.exit(f"missing Std at '{std}'; set STD to a checkout's style directory")
    with tempfile.TemporaryDirectory() as work:
        work = pathlib.Path(work)
        subprocess.run(["cp", "-R", str(ROOT / "Voices/styles"), str(work / "styles")], check=True)
        subprocess.run(["cp", "-R", str(std), str(work / "styles" / "Std")], check=True)
        subprocess.run(["cp", str(path), str(work / "doc.md")], check=True)
        ini = f"StylesPath = styles\nMinAlertLevel = suggestion\n\n[*.md]\nBasedOnStyles = Voices, {voice}\n"
        # Same override test.sh applies: Brevity's required section is the
        # colon reveal the core forbids, and the config is where that is
        # settled. Counting without it would measure a conflict, not a voice.
        extra = ROOT / f"fixtures/after/{voice}.ini"
        if extra.exists():
            ini += extra.read_text()
        (work / ".vale.ini").write_text(ini)
        r = subprocess.run(
            ["vale", "--output=line", "--no-global", "doc.md"],
            cwd=work, capture_output=True, text=True,
        )
        return r.stdout


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--backend", choices=("tiktoken", "anthropic"), default="tiktoken")
    ap.add_argument("--model", default=DEFAULT_MODEL)
    ap.add_argument("--voice", default="Direct")
    args = ap.parse_args()

    skill = ROOT / "script/tokens/no-ai-slop.SKILL.md"
    if not skill.exists():
        sys.exit(f"missing {skill}; fetch it from {SKILL_URL}")

    texts = {
        "prompt": skill.read_text(),
        # What step 4 actually pastes: the shared core plus one voice, the
        # same pairing as `BasedOnStyles = Voices, <voice>`.
        "brief": (ROOT / "briefs/Core.md").read_text()
        + (ROOT / f"briefs/{args.voice}.md").read_text(),
        "alerts_dirty": vale(args.voice, ROOT / "fixtures/before.md"),
        "alerts_clean": vale(args.voice, ROOT / f"fixtures/after/{args.voice}.md"),
    }

    counts, tokenizer = (
        count_anthropic(texts, args.model)
        if args.backend == "anthropic"
        else count_tiktoken(texts)
    )

    print(f"tokenizer: {tokenizer}")
    print(f"voice:     {args.voice}\n")
    labels = {
        "prompt": "no-ai-slop SKILL.md, as loaded",
        "brief": f"briefs/Core.md + briefs/{args.voice}.md",
        "alerts_dirty": "alerts on fixtures/before.md",
        "alerts_clean": f"alerts on fixtures/after/{args.voice}.md",
    }
    for k, label in labels.items():
        print(f"{label:<40} {counts[k]:>6,}")
    print()
    print(json.dumps({"tokenizer": tokenizer, "voice": args.voice, "counts": counts}))


if __name__ == "__main__":
    main()
