#!/usr/bin/env bash
# Bootstrap local developer tooling (macOS / Linux / Git Bash).
# Windows PowerShell users: scripts/setup-dev.ps1
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

echo "==> settle: developer setup"
echo "    repo: ${ROOT}"

resolve_python() {
  if command -v python3 >/dev/null 2>&1; then
    command -v python3
    return
  fi
  if command -v python >/dev/null 2>&1; then
    command -v python
    return
  fi
  echo "error: Python 3 is required (python3 or python on PATH)" >&2
  exit 1
}

PYTHON_BIN="$(resolve_python)"
echo "==> using Python: ${PYTHON_BIN}"

ensure_path() {
  export PATH="${ROOT}/.tools/bin:${HOME}/.local/bin:${PATH}"
}

# Always expose `python` for hooks (Windows-style name) via shim.
install_python_shim() {
  mkdir -p "${ROOT}/.tools/bin"
  cat > "${ROOT}/.tools/bin/python" <<EOF
#!/usr/bin/env bash
exec "${PYTHON_BIN}" "\$@"
EOF
  chmod +x "${ROOT}/.tools/bin/python"
}

install_pre_commit() {
  ensure_path
  install_python_shim

  if command -v pre-commit >/dev/null 2>&1; then
    echo "==> pre-commit already on PATH: $(command -v pre-commit)"
    return
  fi

  if command -v pipx >/dev/null 2>&1; then
    echo "==> installing pre-commit with pipx"
    pipx install pre-commit
    ensure_path
    return
  fi

  if command -v apt-get >/dev/null 2>&1; then
    echo "==> tip: sudo apt-get install -y pipx && pipx ensurepath"
  fi

  TOOLS_VENV="${ROOT}/.tools/pre-commit-venv"
  echo "==> installing pre-commit into ${TOOLS_VENV}"
  "${PYTHON_BIN}" -m venv "${TOOLS_VENV}"
  # shellcheck disable=SC1091
  source "${TOOLS_VENV}/bin/activate"
  python -m pip install --upgrade pip
  python -m pip install pre-commit
  ln -sfn "${TOOLS_VENV}/bin/pre-commit" "${ROOT}/.tools/bin/pre-commit"
  ensure_path
  deactivate || true

  if ! command -v pre-commit >/dev/null 2>&1; then
    echo "error: pre-commit installed but not on PATH; add ${ROOT}/.tools/bin to PATH" >&2
    exit 1
  fi
}

install_pre_commit
ensure_path
install_python_shim

echo "==> installing git hooks (pre-commit, commit-msg, pre-push) — once per clone"
pre-commit install --hook-type pre-commit
pre-commit install --hook-type commit-msg
pre-commit install --hook-type pre-push

chmod +x scripts/setup-dev.sh \
         scripts/enable-branch-protection.sh \
         scripts/git-hooks/commit_msg.py \
         scripts/git-hooks/pre_push.py \
         scripts/git-hooks/which_python.py 2>/dev/null || true

echo "==> running pre-commit once on all files (may auto-fix)"
SKIP=no-commit-to-branch,conventional-commit-message,block-push-to-protected-branches \
  pre-commit run --all-files || true

cat <<'EOF'

Setup complete. You do NOT need to re-run this on every commit — only after a fresh clone.

Next steps:
  1. git switch -c feat/short-description
  2. git commit -m "feat: describe the change"
  3. git push -u origin HEAD   # then open a PR
  4. Enable remote protection (one-time per GitHub repo):
       ./scripts/enable-branch-protection.sh

Docs: docs/WORKFLOW.md
EOF
