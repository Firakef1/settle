#!/usr/bin/env python3
"""Block direct / force pushes to main and master. Cross-platform."""
from __future__ import annotations

import os
import re
import subprocess
import sys

PROTECTED = re.compile(r"^refs/heads/(main|master)$")
ZERO = re.compile(r"^0+$")


def _parent_is_force() -> bool:
    """Best-effort: detect -f / --force* on the parent git push command."""
    try:
        import psutil  # type: ignore  # optional

        parent = psutil.Process(os.getppid())
        cmd = " ".join(parent.cmdline())
    except Exception:
        try:
            out = subprocess.check_output(
                ["ps", "-o", "args=", "-p", str(os.getppid())],
                text=True,
                stderr=subprocess.DEVNULL,
            )
            cmd = out.strip()
        except Exception:
            return False
    return bool(
        re.search(r"(^|\s)(--force|-f|--force-with-lease|--force-if-includes)(\s|$)", cmd)
    )


def _is_ancestor(older: str, newer: str) -> bool:
    try:
        subprocess.check_call(
            ["git", "merge-base", "--is-ancestor", older, newer],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        return True
    except subprocess.CalledProcessError:
        return False
    except FileNotFoundError:
        return True  # fail open if git missing (should not happen in a hook)


def _fail_direct(branch: str) -> None:
    print(
        f"""pre-push: refusing to push directly to '{branch}'.

Protected branches: main, master

Do this instead:
  1. git switch -c feat/short-description
  2. git push -u origin HEAD
  3. open a pull request into main

Emergency override (local only — still blocked if GitHub branch protection is on):
  ALLOW_PUSH_PROTECTED=1 git push ...

See docs/WORKFLOW.md""",
        file=sys.stderr,
    )


def _fail_force(ref: str) -> None:
    print(
        f"""pre-push: refusing force-push / non-fast-forward update to '{ref}'.

Never rewrite history on main/master. Open a revert PR instead.

Emergency override (local only):
  ALLOW_FORCE_PROTECTED=1 git push ...

See docs/WORKFLOW.md""",
        file=sys.stderr,
    )


def main() -> int:
    forceish = _parent_is_force()
    allow_push = os.environ.get("ALLOW_PUSH_PROTECTED") == "1"
    allow_force = os.environ.get("ALLOW_FORCE_PROTECTED") == "1"

    for line in sys.stdin:
        parts = line.strip().split()
        if len(parts) < 4:
            continue
        _local_ref, local_sha, remote_ref, remote_sha = parts[:4]

        if ZERO.match(local_sha):
            continue  # remote branch delete
        if not PROTECTED.match(remote_ref):
            continue

        branch = remote_ref.rsplit("/", 1)[-1]

        if not allow_push:
            _fail_direct(branch)
            return 1
        print(
            f"pre-push: ALLOW_PUSH_PROTECTED=1 set — allowing direct push to {branch}",
            file=sys.stderr,
        )

        is_nff = False
        if not ZERO.match(remote_sha) and not _is_ancestor(remote_sha, local_sha):
            is_nff = True

        if forceish or is_nff:
            if allow_force:
                print(
                    f"pre-push: ALLOW_FORCE_PROTECTED=1 set — allowing history rewrite on {branch}",
                    file=sys.stderr,
                )
                continue
            _fail_force(remote_ref)
            return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
