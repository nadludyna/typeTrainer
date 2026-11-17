# How to work on typeTrainer (for AI agents)

## What to read first

This file says how to work on the project. Everything else has its own file:

| Before you... | Read |
|---|---|
| start any task | `docs/PRODUCT.md`: what the product is, its principles, what is not built now |
| change code | `docs/CODE-RULES.md`: the stack, the architecture rules, comments |
| need to know where something is | `docs/HOW-THE-CODE-WORKS.md`: packages, modules, the API, the database |
| touch Learn or Reinforce | `docs/HOW-THE-MODES-WORK.md`: how the trainer picks what you type |
| change the interface | `docs/DESIGN-RULES.md`: how the screen looks and behaves |
| write any text the user sees | `docs/TEXT-RULES.md` |
| run, test or measure | `CONTRIBUTING.md` |

Where they differ, this file wins, then `docs/PRODUCT.md`, then the others.

If `AGENTS.local.md` exists next to this file, read it too: it holds the
personal working rules of whoever runs this copy (language, how they test,
their words for commands). It is not in git, and where it differs from this
file, it wins.

## Working with the maintainer

- Code, comments, commit messages and docs: English.
- Work in small steps. For anything bigger than one feature, show a short plan
  first and wait for an OK.
- After every task, explain in plain words what you changed and why.
- Commits in your own branch are fine. `git push` and pull requests only when
  the maintainer says so.
- `main` is protected: changes reach it only through a pull request, merged by
  the maintainer. Never push to `main` directly.

## How we work

The plan is issue #305: tasks go one at a time, top down, each its own issue.

1. **First an opinion, then work.** On any request or mockup say what you think
   and what you would do better, and why. Build after the OK. An idea outside
   the request is a proposal, not an action.
2. **One branch for one topic, one PR at the end.** A GitHub issue is not
   required.
3. **Every PR description answers three things:** (a) what changed;
   (b) what else it affects (other modes, combinations of settings);
   (c) how it was checked: what was run, on what, with which numbers.
4. **A change to the typing screen is measured** before the PR, against
   `main`, with the numbers in the PR, and must not be worse. The canonical
   measurement: `node tools/browser/perf.mjs` (CONTRIBUTING.md): style recalc
   per key at full speed, key → caret start and the share of slow frames with
   the CPU slowed 6×. The goal: under 2 ms of style recalc per key for the
   screen itself (`perf.mjs --bare`), caret start ≤ 40 ms and slow frames
   ≤ 15% with everything on.
5. **Measure first, then change numbers.** No tuning without a measurement.
6. **Comments and docs never retell history.** They say what is true now;
   history lives in git.
7. **No tool for a needed check is not a reason to skip the check.** Say so.
8. **Docs do not multiply.** Every rule and fact has one home in the files
   listed under "What to read first"; other places link to it. No new files
   in `docs/` or the repo root without the maintainer's decision.
9. **Nothing personal in the repo.** Anyone who clones or forks it gets a
   working project with no notes about one person's life, studies or setup.
   Those belong in `AGENTS.local.md` or outside the repo.

Beyond orders an engineer's position is expected: point out when a change has
consequences outside its issue, offer simpler paths, say no with reasons
when a task leads somewhere worse. Initiative means a proposal; the scope
is always the maintainer's decision.

## Before you say a task is done

1. The checks in `CONTRIBUTING.md` ("Test it", "Browser checks") all pass.
2. New engine logic has unit tests, including edge cases (no data yet, zero values).
3. Docs updated if behaviour or the API changed.
4. An interface change is checked in a browser at desktop width, and the
   stats screen also at phone width (`docs/DESIGN-RULES.md`, section 3).
5. A short summary: what changed, which files, how to check it in the browser.
