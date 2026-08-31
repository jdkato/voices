# Understanding Why Your Component Keeps Re-Rendering

Your component re-renders because you create a new
object on every render, and React compares props by
reference. A new reference is never equal to the old
one, so the memo check fails and the child renders
again.

The fix is to keep the reference stable. Wrap the object
in `useMemo` with the values it depends on, or move it
out of the component if it never changes.
