# settle

Team repository for Settle.

## Start here

Language/stack is not chosen yet. Commit, branch, and PR guardrails are ready on macOS, Linux, and Windows.

```bash
# macOS / Linux / Git Bash
./scripts/setup-dev.sh

# Windows PowerShell
powershell -ExecutionPolicy Bypass -File scripts/setup-dev.ps1
```

Read **[docs/WORKFLOW.md](docs/WORKFLOW.md)** for conventions and how to enable GitHub branch protection (remote backstop).

One-time remote lock (needs `gh` admin access):

```bash
./scripts/enable-branch-protection.sh
# or: powershell -File scripts/enable-branch-protection.ps1
```
