# Flock Worktree Environment

## CRITICAL: Git Command Restrictions

You are working in a flock-managed worktree. The following restrictions are enforced:

- **DO NOT** run `git checkout` or `git switch` — your branch (fix/issue-66-tcy-staking-unstaking-skill) is fixed.
- **DO NOT** run `git worktree add/remove/move/prune` — flock manages worktrees.
- All other git commands (add, commit, push, status, diff, log, etc.) are allowed.

## Environment Setup

Before running any git command, ensure the flock git wrapper is on your PATH:

`
export PATH="/Users/raghavsood/.flock/state/github.com/vultisig/mcp/worktrees/fix/issue-66-tcy-staking-unstaking-skill/.flock/bin:$PATH"
`

Or prefix individual commands:

`
PATH="/Users/raghavsood/.flock/state/github.com/vultisig/mcp/worktrees/fix/issue-66-tcy-staking-unstaking-skill/.flock/bin:$PATH" git status
`

This wrapper enforces worktree isolation and prevents accidental branch switches or worktree modifications.

## Worktree Details
- **Path**: /Users/raghavsood/.flock/state/github.com/vultisig/mcp/worktrees/fix/issue-66-tcy-staking-unstaking-skill
- **Branch**: fix/issue-66-tcy-staking-unstaking-skill
- **Do NOT** remove this worktree — flock manages cleanup.
