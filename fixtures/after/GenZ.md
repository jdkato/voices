# Understanding Why Your Component Keeps Re-Rendering

Your render path is cooked. You build a new object every
render, and React compares props by reference. A new
reference never equals the old one, so the memo check
fails and the child renders again.

Wrap the object in `useMemo` with the values it depends
on. Stable reference, clean diff, massive W.
