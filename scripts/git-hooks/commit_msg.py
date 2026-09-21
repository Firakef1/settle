#!/usr/bin/env python3
"""Validate commit messages (Conventional Commits). Cross-platform."""
from __future__ import annotations

import os
import re
import sys

ALLOWED = (
    "feat",
    "fix",
    "docs",
    "style",
    "refactor",
    "perf",
    "test",
    "build",
    "ci",
    "chore",
    "revert",
)

SUBJECT_RE = re.compile(
    rf"^({'|'.join(ALLOWED)})"
    r"(\([a-z0-9][a-z0-9._/-]*\))?"
    r"(!)?"
    r": .+$"
)

GIT_SPECIAL = re.compile(r"^(Merge|Revert|fixup!|squash!) ")


def main() -> int:
    if len(sys.argv) < 2 or not os.path.isfile(sys.argv[1]):
        print("commit-msg: missing commit message file", file=sys.stderr)
        return 1

    raw = open(sys.argv[1], encoding="utf-8").read().splitlines()
    lines = [ln.rstrip() for ln in raw if not ln.startswith("#")]
    while lines and not lines[0].strip():
        lines.pop(0)
    subject = lines[0] if lines else ""

    if not subject.strip():
        print("commit-msg: empty commit message", file=sys.stderr)
        return 1

    if GIT_SPECIAL.match(subject):
        return 0

    if not SUBJECT_RE.match(subject):
        print(
            """commit-msg: subject must follow Conventional Commits.

  <type>(optional-scope)!: <description>

Examples:
  feat: add invoice PDF export
  fix(auth): handle expired refresh tokens
  docs: explain branch protection setup
  chore!: drop legacy settings file

Allowed types:
  feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert

Rules:
  - lowercase type
  - description after ": " (colon + space)
  - subject line ideally ≤ 72 characters
  - no trailing period on the subject

See docs/WORKFLOW.md""",
            file=sys.stderr,
        )
        return 1

    if len(subject) > 100:
        print(
            f"commit-msg: subject is {len(subject)} chars; keep it ≤ 100 (prefer ≤ 72)",
            file=sys.stderr,
        )
        return 1

    if subject.endswith("."):
        print("commit-msg: do not end the subject with a period", file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
