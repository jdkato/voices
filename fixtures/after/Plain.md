# Understanding Why Your Component Keeps Re-Rendering

Your component re-renders because you create a new
object on every render. React compares props by
reference, and a new reference is never equal to the old
one. The memo check fails, so the child renders again.

Wrap the object in `useMemo` with the values it depends
on. If it never changes, move it out of the component.
