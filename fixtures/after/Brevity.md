# New object, new reference

Your component re-renders because you build a new object
every render. React compares props by reference. A new
reference never equals the old one.

Why it matters: the memo check fails, so every child re-
renders on every parent render. On a large tree that is
the whole frame budget.

## The fix

Wrap the object in `useMemo` with the values it depends
on. If it never changes, move it out of the component.
