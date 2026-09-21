#!/usr/bin/env python3
"""Resolve a Python interpreter the same way setup scripts do."""
from __future__ import annotations

import os
import shutil
import sys


def resolve() -> str:
    for name in ("python", "python3", "py"):
        path = shutil.which(name)
        if not path:
            continue
        if name == "py":
            return path  # caller should use `py -3`
        return path
    raise SystemExit("error: Python 3 is required (python / python3 / py)")


if __name__ == "__main__":
    print(resolve())
