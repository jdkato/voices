#!/bin/sh
#
# One draft, nine voices. For each voice, `fixtures/before.md` is checked with
# `Voices, <voice>` enabled and the alerts compared to a golden file; then the
# rewrite in `fixtures/after/<voice>.md` is checked the same way and required
# to produce nothing at all.
#
# The second half is the load-bearing one. A Vale rule that matches nothing
# fails silently -- it loads, runs, and reports success -- so "no alerts" only
# means something when a paired fixture proves the rules fire. Three rules in
# this package were dead on arrival: a `raw:` list concatenates its entries
# rather than alternating them, `metric` has no readability formulas, and a
# token list written in the infinitive never matches the past tense.
#
# `./test.sh -u` rewrites the golden files instead of comparing.
set -eu

update=0
[ "${1:-}" = "-u" ] && update=1

root=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

cp -R "$root/Voices/styles" "$work/styles"
mkdir -p "$root/testdata"

voices="Direct Plain Unslop Brevity Simple GenZ"

run() { # <file> -> alerts on stdout
	cp "$1" "$work/doc.md"
	(cd "$work" && vale --output=line --no-global doc.md 2>&1 || true)
}

status=0
for voice in Voices $voices; do
	# `Voices` is the shared core and is always on; naming it twice is
	# harmless and keeps the loop uniform.
	cat > "$work/.vale.ini" <<INI
StylesPath = styles
MinAlertLevel = suggestion

[*.md]
BasedOnStyles = Voices, $voice
INI
	# Voices can conflict: Smart Brevity's "Why it matters:" is the colon
	# reveal the shared core forbids. An optional .ini fragment beside the
	# rewrite settles it the way a user would, in the config.
	extra="$root/fixtures/after/$voice.ini"
	[ -f "$extra" ] && cat "$extra" >> "$work/.vale.ini"

	got=$(run "$root/fixtures/before.md")
	golden="$root/testdata/$voice.before.txt"
	if [ "$update" -eq 1 ]; then
		printf '%s\n' "$got" > "$golden"
	elif [ ! -f "$golden" ]; then
		echo "FAIL $voice/before: no golden file; run ./test.sh -u"
		status=1
	elif [ "$got" != "$(cat "$golden")" ]; then
		echo "FAIL $voice/before"
		printf '%s\n' "$got" | diff -u "$golden" - || true
		status=1
	else
		echo "ok   $voice/before ($(printf '%s' "$got" | grep -c . || true) alerts)"
	fi

	after="$root/fixtures/after/$voice.md"
	if [ ! -f "$after" ]; then
		echo "FAIL $voice/after: $after is missing"
		status=1
		continue
	fi
	got=$(run "$after")
	if [ -n "$got" ]; then
		echo "FAIL $voice/after: the rewrite still violates the voice"
		printf '%s\n' "$got"
		status=1
	else
		echo "ok   $voice/after (clean)"
	fi
done

[ "$update" -eq 1 ] && echo "golden files rewritten"
exit $status
