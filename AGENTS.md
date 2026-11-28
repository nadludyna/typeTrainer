Be concise. Code, comments, commits: English.
Run: `go run .` → http://localhost:3000. `DATABASE_URL=postgres://…` keeps data in PostgreSQL, else memory.
Check before a PR: `go run ./tools/check`. It runs every linter and test; what it enforces is not repeated here.
Every tunable number lives in `internal/config`. Comments say why, never history: history is git.
A change to the typing screen is measured against `main` before the PR: `node tools/perf/perf.mjs --against main`.
A change to how a mode picks words: show `go run ./tools/sim` before and after.
Ask before: a new dependency, a change to a mode's algorithm, cutting or adding a feature.
One topic, one branch from fresh `main`, one PR. Push and PRs only when the maintainer says so; never push to `main`.
PR body: what changed, what else it touches, how it was checked (commands and numbers).
`AGENTS.local.md`, if present, holds the maintainer's personal rules and wins over this file.
