# Developer workflow & guardrails

---

## Quick start (once per clone)

**macOS / Linux / Git Bash:**

```bash
./scripts/setup-dev.sh
```

**Windows (PowerShell):**

```powershell
powershell -ExecutionPolicy Bypass -File scripts/setup-dev.ps1
```

That installs `pre-commit` if needed, registers normal git hooks (`pre-commit`, `commit-msg`, `pre-push`), and runs a first check.

You do **not** re-run setup on every commit — only after a fresh clone (or if `.git/hooks` were wiped).

Then branch before working:

```bash
git switch -c feat/short-description
```

---

## Do I need to run the script every time?

No. Setup writes into `.git/hooks/`. After that, plain `git commit` / `git push` triggers the guards automatically.

| When | Action |
|------|--------|
| First clone / new machine | Run `setup-dev.sh` or `setup-dev.ps1` once |
| Every commit / push | Nothing — git runs hooks |
| Teammate skips hooks / never ran setup | Remote CI + branch protection still catch mistakes |

---

## Windows support

| Piece | Works on Windows? |
|-------|-------------------|
| `pre-commit` + shared config | Yes (needs Python 3 on PATH) |
| Policy hooks (`commit_msg.py`, `pre_push.py`) | Yes — Python, not bash |
| `setup-dev.ps1` | Yes — native PowerShell bootstrap |
| GitHub Actions | Same for everyone (runs on GitHub) |
| `setup-dev.sh` | Use Git Bash or WSL, or use `.ps1` instead |

Install Python from [python.org](https://www.python.org/downloads/) and enable **Add to PATH**.

---

## What was added

| Path | Purpose |
|------|---------|
| `.pre-commit-config.yaml` | Shared hook config |
| `scripts/git-hooks/commit_msg.py` | Conventional Commits (local + reused in CI) |
| `scripts/git-hooks/pre_push.py` | Block direct / force push to `main`/`master` |
| `scripts/setup-dev.sh` / `setup-dev.ps1` | One-time bootstrap per OS |
| `scripts/enable-branch-protection.sh` / `.ps1` | Turn on GitHub ruleset / branch protection |
| `.github/workflows/repo-hygiene.yml` | CI: pre-commit, PR title/branch rules, commit messages |
| `.github/workflows/protect-default-branch.yml` | CI alarm on non-PR updates to `main` |
| `.editorconfig` / `.gitattributes` / `.gitignore` | Consistent files; keep secrets out |
| PR + issue templates | Checklist and intake forms |

---

## Protection layers

```text
Local hooks  →  fast feedback on your machine
     ↓ (if skipped / missing)
CI on PRs    →  pre-commit + title + branch name + commit messages
     ↓
GitHub ruleset / branch protection  →  real block on direct push to main
     ↓ (if someone still got a push through before rules existed)
protect-default-branch workflow     →  loud failure / visible mistake
```

### Local

1. **No commits on `main`/`master`** — `no-commit-to-branch`
2. **No direct / force push to `main`/`master`** — `pre_push.py`
3. **Conventional commit subjects** — `commit_msg.py`
4. **File hygiene** — whitespace, secrets, large files, YAML/JSON, Actions lint, LF

Escape hatches (local only; remote rules still apply once enabled):

```bash
ALLOW_PUSH_PROTECTED=1 git push ...
ALLOW_FORCE_PROTECTED=1 git push ...
```

### Remote (CI)

On every pull request into `main`/`master`:

- Same file hygiene via `pre-commit`
- Conventional Commits **PR title**
- Branch name like `feat/…`, `fix/…`, `chore/…`
- **Every commit** in the PR re-checked with `commit_msg.py` (covers `--no-verify`)
- Non-draft PRs cannot have WIP / “do not merge” titles

On push to `main`/`master`:

- `no direct push` job fails if the commit is not tied to a pull request (alarm; cannot rewind the push)

### Remote (must enable once)

Local hooks and CI are not enough to **prevent** a direct `git push origin main`. Enable GitHub protection:

```bash
# macOS / Linux / Git Bash
./scripts/enable-branch-protection.sh

# Windows PowerShell
powershell -File scripts/enable-branch-protection.ps1
```

Requires [`gh`](https://cli.github.com/) (`gh auth login`) and **admin** on the repo.

Optional: require one approving review:

```bash
REQUIRED_APPROVALS=1 ./scripts/enable-branch-protection.sh
```

That creates a ruleset (or classic branch protection) which:

- Requires a pull request
- Requires status checks: `pre-commit`, `PR conventions`, `commit messages`
- Blocks force-pushes and branch deletion
- Requires conversation resolution

**Manual path:** GitHub → Settings → Rules → Rulesets (or Branches → Branch protection) for `main`.

---

## Commit & branch conventions

**Commit / PR title:**

```text
<type>(optional-scope)!: <description>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

```bash
git commit -m "feat: add settlement export endpoint"
git commit -m "fix(auth): handle expired refresh tokens"
```

**Branch names:** `<type>/<short-description>` — e.g. `feat/invoice-export`, `chore/enable-branch-protection`

---

## Daily workflow

```text
main (protected)
  └── feat/my-change   ← commit & push here
        └── Pull Request → main   ← review + CI → merge
```

1. `git switch main && git pull`
2. `git switch -c feat/my-change`
3. Edit → `git add` → `git commit -m "feat: …"`
4. `git push -u origin HEAD`
5. Open a PR (Conventional Commits title)
6. Merge only after checks pass

### First push of `main` to a new remote

Prefer pushing a feature branch + PR. If you must push `main` once before rulesets exist:

```bash
ALLOW_PUSH_PROTECTED=1 git push -u origin main
```

Then run `enable-branch-protection` immediately.

---

## Intentionally not included yet

Stack undecided — so no ESLint/Ruff/gofmt, no app test/build pipelines, no lockfiles, no Docker app stack. Add those beside these guardrails when you pick a language.

---

## Useful commands

```bash
pre-commit run --all-files    # re-check everything
pre-commit autoupdate         # bump hook versions
git commit --no-verify        # skip local hooks (CI still runs)
```

---

## Review notes

Nothing from this setup was committed or pushed by the agent. After you review:

1. Run setup if needed (`setup-dev.sh` / `setup-dev.ps1`)
2. Leave `main`: `git switch -c chore/add-workflow-guardrails`
3. Commit, push, open a PR
4. Run `./scripts/enable-branch-protection.sh` (or `.ps1`) on the GitHub remote
