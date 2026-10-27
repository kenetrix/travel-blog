# Travel Blog

Go backend (`backend/`) and Vue 3 + TypeScript frontend (`frontend/`, Vite, Pinia, MapLibre).

## Run locally

From the repo root: `npm install` once (also installs `frontend/`), then
`npm run dev` starts the backend (`:8080`) and frontend (`:5173`) together.

## Working style

- Do not summarize at the end of a task; just confirm the implementation is done.
- Keep output short. Do not narrate your thinking.
- Comment only what the code does not say (why, constraints, edge cases), in at most two lines. No comments that repeat a name, e.g. `// CookieName is the name of the cookie`.
- Format frontend code with Prettier (`npx prettier --write <files>` in `frontend/`).

## Language

Everything in the repository is written in English: code comments, identifiers,
documentation (READMEs, `docs/`, OpenSpec artifacts), UI text, error messages,
branch names and commit messages. Do not add any other language
text; translate it when touching a file that still has some.

## Database

Until the `v1.0.0` tag exists, do not add migration scripts. Change the files in
`backend/migrations/data/001_init/` and recreate the test database. From `v1.0.0` on,
`001_init` is frozen: add new numbered migrations (`002_<name>.sql`, ...) instead.

## Releases

Pushing a `v*` tag builds a GitHub release (`.github/workflows/release.yml`), which deletes
every previous release, so only the newest one exists.

- Never move or re-push an existing tag. Every release gets a new version.
- Before `v1.0.0`, release as the next `0.x.y`: bump `y` for fixes only, `x` when features
  are included (e.g. `v0.1.0` → `v0.1.1` or `v0.2.0`). Check `git tag -l` for the latest.
- Tag the commit with `git tag -a v0.x.y -m "v0.x.y"`, then push only that tag.

## Branches

Until the `v1.0.0` tag exists (`git tag -l v1.0.0` prints nothing), develop directly on `main`
and skip the branching rules below. From `v1.0.0` on, they apply.

Every feature or fix is developed in its own branch, named:

```
<type>/<feature_or_fix_name>/<optional>
```

- `type` is `feat` or `fix`:
  - new domain or feature → `feat`
  - change to an existing feature → `fix`
- `feature_or_fix_name` is one word, two at most, summarizing what was implemented or changed.
- Add the trailing `optional` part only if the work could become very large; avoid it otherwise.

Process:

1. Run `git status` to check whether you are already on the correct branch.
2. If a new branch is needed, first check the remote for a relevant existing branch in the same context and check that out. Otherwise create a new one.
3. Report the final branch name.

## Commits

Every commit message follows [Conventional Commits](https://www.conventionalcommits.org):

```
<type>[optional scope]: <short imperative summary>

[optional body or reasoning]
```

- `type` is one of `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`.
- Add a scope when it makes sense, e.g. `feat(frontend): ...` or `chore(repo): ...`.
- Summary in imperative mood, no trailing period, under ~72 characters.
- Choose the type by what the change does:
  - initial commit of a repo → `chore`
  - new feature → `feat`
  - bug fix → `fix`
  - documentation only → `docs`
- Check the message against this schema before committing; a message that violates it is a failed task.

Process:

1. Run `git status` and `git log --oneline` to understand the current state.
2. Confirm the branch is correct and matches the context.
3. Stage the intended files (`git add <paths>`). Group logically related changes; split unrelated changes into separate commits.
4. Write the message following the schema above.
5. Commit, then report the final message, short hash and file count.

Keep commits atomic: if the working tree contains multiple logical changes, split them (e.g. `feat(frontend): ...` and `feat(backend): ...`) instead of one generic commit.
