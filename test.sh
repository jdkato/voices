#!/bin/sh
#
# One draft, six voices. For each voice, `fixtures/before.md` is checked with
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
# `fixtures/guards/` then pins the negative half: the constructions each rule
# must leave alone, checked in the same way.
#
# `./test.sh -u` rewrites the golden files instead of comparing.
set -eu

update=0
[ "${1:-}" = "-u" ] && update=1
status=0

root=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

cp -R "$root/Voices/styles" "$work/styles"
mkdir -p "$root/testdata"

voices="Direct Plain Unslop Brevity Simple GenZ"

# Alerts that share a line and column come back in whatever order the checks
# ran, and that order is not part of the contract -- it has changed between
# Vale releases and would fail this suite on a build where nothing about the
# rules moved. Sorting by line, then column, then rule name compares the set
# of alerts, which is what the fixtures are actually asserting.
run() { # <file> -> alerts on stdout, sorted
	cp "$1" "$work/doc.md"
	(cd "$work" && vale --output=line --no-global doc.md 2>&1 || true) |
		sort -t: -k2,2n -k3,3n -k4,4
}

# compare checks one fixture against its golden file, or rewrites the golden
# under `-u`. It sets `status` rather than exiting, so one bad voice does not
# hide the rest.
compare() { # <label> <fixture> <golden>
	_got=$(run "$2")
	if [ "$update" -eq 1 ]; then
		printf '%s\n' "$_got" > "$3"
		return 0
	fi
	if [ ! -f "$3" ]; then
		echo "FAIL $1: no golden file; run ./test.sh -u"
		status=1
	elif [ "$_got" != "$(cat "$3")" ]; then
		echo "FAIL $1"
		printf '%s\n' "$_got" | diff -u "$3" - || true
		status=1
	else
		echo "ok   $1 ($(printf '%s' "$_got" | grep -c . || true) alerts)"
	fi
}

# The GenZ slang list is counted three ways -- per sentence, per paragraph, and
# for presence -- and Vale has no include, so the token lives in three files.
# Drift there is silent in the worst way: Presence would pass on a term Density
# no longer counts. Compare the copies instead of trusting them.
tokens=$(grep -h '^token:' "$root"/Voices/styles/GenZ/*.yml | sort -u | wc -l)
if [ "$tokens" -ne 1 ]; then
	echo "FAIL GenZ: the slang token differs across Density, Budget and Presence"
	grep -n '^token:' "$root"/Voices/styles/GenZ/*.yml
	exit 1
fi

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

	compare "$voice/before" "$root/fixtures/before.md" "$root/testdata/$voice.before.txt"

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

# The guard fixtures pin the other half of every rule: what it must leave
# alone. `before.md` cannot do this job -- it holds no perfect passive, no
# hyphenated slang and no admonition label -- so a rule that quietly widened
# would keep passing. Each guard file mixes the constructions that must fire
# with the ones that must not, and the golden records both.
for pair in Prose:Voices,Plain,Unslop,Brevity GenZ:GenZ; do
	name=${pair%%:*}
	cat > "$work/.vale.ini" <<INI
StylesPath = styles
MinAlertLevel = suggestion

[*.md]
BasedOnStyles = $(echo "${pair#*:}" | sed 's/,/, /g')
INI
	compare "guards/$name" "$root/fixtures/guards/$name.md" "$root/testdata/$name.guards.txt"
done

[ "$update" -eq 1 ] && echo "golden files rewritten"
exit $status
