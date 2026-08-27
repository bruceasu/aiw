# Use the primary workspace by default

AIW Tasks use the primary Git checkout and its current branch by default;
linked worktrees are created only through an explicitly authorized isolation
transition. This separates Task lifecycle from Git delivery and avoids fixed
branch/worktree overhead for ordinary sequential changes while retaining
isolation for parallel, conflicting, long-running, or disposable work.
