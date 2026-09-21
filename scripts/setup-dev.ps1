# Bootstrap local developer tooling on Windows (PowerShell).
# macOS / Linux / Git Bash users: ./scripts/setup-dev.sh
#Requires -Version 5.1
$ErrorActionPreference = "Stop"

$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $Root

Write-Host "==> settle: developer setup (Windows)"
Write-Host "    repo: $Root"

function Resolve-Python {
  foreach ($name in @("python", "python3", "py")) {
    $cmd = Get-Command $name -ErrorAction SilentlyContinue
    if (-not $cmd) { continue }
    if ($name -eq "py") {
      return @{ Exe = $cmd.Source; Args = @("-3") }
    }
    # Prefer real Python, skip Windows Store stub if it fails --version
    try {
      & $cmd.Source --version 2>$null | Out-Null
      if ($LASTEXITCODE -eq 0) {
        return @{ Exe = $cmd.Source; Args = @() }
      }
    } catch { }
  }
  throw "Python 3 is required. Install from https://www.python.org/downloads/ and enable 'Add to PATH'."
}

$Py = Resolve-Python
Write-Host "==> using Python: $($Py.Exe) $($Py.Args -join ' ')"

$ToolsBin = Join-Path $Root ".tools\bin"
New-Item -ItemType Directory -Force -Path $ToolsBin | Out-Null

# Shim so hooks can call `python` consistently
$shim = Join-Path $ToolsBin "python.cmd"
$pyArgs = ($Py.Args + @("%*")) -join " "
@"
@echo off
"$($Py.Exe)" $pyArgs
"@ | Set-Content -Path $shim -Encoding ASCII

$env:PATH = "$ToolsBin;$env:LOCALAPPDATA\Programs\Python\Scripts;$env:PATH"

function Test-PreCommit {
  return [bool](Get-Command pre-commit -ErrorAction SilentlyContinue)
}

if (-not (Test-PreCommit)) {
  $venv = Join-Path $Root ".tools\pre-commit-venv"
  Write-Host "==> installing pre-commit into $venv"
  & $Py.Exe @($Py.Args) -m venv $venv
  $venvPy = Join-Path $venv "Scripts\python.exe"
  & $venvPy -m pip install --upgrade pip
  & $venvPy -m pip install pre-commit
  $preCommit = Join-Path $venv "Scripts\pre-commit.exe"
  Copy-Item -Force $preCommit (Join-Path $ToolsBin "pre-commit.exe")
  $env:PATH = "$ToolsBin;$env:PATH"
}

if (-not (Test-PreCommit)) {
  throw "pre-commit installed but not on PATH; add $ToolsBin to PATH"
}

Write-Host "==> installing git hooks (pre-commit, commit-msg, pre-push) — once per clone"
pre-commit install --hook-type pre-commit
pre-commit install --hook-type commit-msg
pre-commit install --hook-type pre-push

Write-Host "==> running pre-commit once on all files (may auto-fix)"
$env:SKIP = "no-commit-to-branch,conventional-commit-message,block-push-to-protected-branches"
try {
  pre-commit run --all-files
} catch {
  Write-Host "pre-commit reported issues (often auto-fixed); re-run if needed."
}

Write-Host @"

Setup complete. You do NOT need to re-run this on every commit — only after a fresh clone.

Next steps:
  1. git switch -c feat/short-description
  2. git commit -m "feat: describe the change"
  3. git push -u origin HEAD
  4. Enable remote protection (one-time per GitHub repo):
       powershell -File scripts/enable-branch-protection.ps1

Docs: docs/WORKFLOW.md
"@
