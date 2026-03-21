# Flock Worktree Environment

## CRITICAL: Git Command Restrictions

You are working in a flock-managed worktree. The following restrictions are enforced:

- **DO NOT** run `git checkout` or `git switch` — your current flock-assigned branch is fixed.
- **DO NOT** run `git worktree add/remove/move/prune` — flock manages worktrees.
- All other git commands (add, commit, push, status, diff, log, etc.) are allowed.

## Environment Setup

Before running any git command, ensure the flock git wrapper is on your PATH:

```bash
export PATH="<worktree_path>/.flock/bin:$PATH"
```

Or prefix individual commands:

```bash
PATH="<worktree_path>/.flock/bin:$PATH" git status
```

This wrapper enforces worktree isolation and prevents accidental branch switches or worktree modifications.

## Worktree Details
- **Path**: `<worktree_path>`
- **Branch**: `<branch_name>`
- **Do NOT** remove this worktree — flock manages cleanup.
