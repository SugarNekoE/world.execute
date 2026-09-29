# Agent Rules

## Code Style

- Do not add comments unless they are strictly necessary. Prefer
  self-explanatory code over explanatory comments.

## Git Conventions

- Commit messages follow Conventional Commits: `type(scope): subject`.
- Use `git commit -s` so the `Signed-off-by:` trailer is present.
- Use a lowercase imperative subject with no trailing period and a concise,
  meaningful scope such as `app`, `ui`, `tracking`, `carrier`, `auth`, `db`,
  `agents`, or `tooling`.
- Never add a `Co-authored-by:` trailer.

## Review

- Every implementation must be reviewed by a human before it is considered
  done. Do not treat work as complete, merge it, or move on to the next task
  until a human has reviewed and accepted it.

## Changes

- Ask the human for a decision before making any change that was not planned.
  When an unplanned change is needed, stop and ask rather than proceeding.
