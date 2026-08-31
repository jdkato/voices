# Understanding Why Your Component Keeps Re-Rendering

Here's the thing: it's worth noting that the reason your
React component is re-rendering is likely because you're
creating a new object reference on each render cycle,
which breaks React's referential equality check — so you
may want to consider memoization.

This is not just a performance problem, it's a
correctness problem. Furthermore, experts agree that a
robust approach to this paradigm shift will empower your
team to streamline a number of things in the render
path, underscoring its significance.

In conclusion, the team made a decision to leverage
several caching solutions.
