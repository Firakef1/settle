# Apply GitHub branch protection / ruleset for main (remote backstop).
# Requires: gh auth login  (admin on the repo)
#Requires -Version 5.1
$ErrorActionPreference = "Stop"

if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
  throw "GitHub CLI (gh) is required — https://cli.github.com/"
}

gh auth status 2>$null | Out-Null
if ($LASTEXITCODE -ne 0) {
  throw "Not logged in — run: gh auth login"
}

$Branch = if ($args.Count -ge 1) { $args[0] } else { "main" }
$Approvals = if ($env:REQUIRED_APPROVALS) { [int]$env:REQUIRED_APPROVALS } else { 0 }
$Repo = gh repo view --json nameWithOwner -q .nameWithOwner

Write-Host "==> enabling protection for ${Repo}@${Branch}"
Write-Host "    required approvals: $Approvals"

$payload = @"
{
  "name": "Protect $Branch",
  "target": "branch",
  "enforcement": "active",
  "conditions": {
    "ref_name": {
      "include": ["refs/heads/$Branch"],
      "exclude": []
    }
  },
  "rules": [
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": $Approvals,
        "dismiss_stale_reviews": true,
        "require_code_owner_review": false,
        "require_last_push_approval": false,
        "required_review_thread_resolution": true
      }
    },
    {
      "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": true,
        "do_not_enforce_on_create": false,
        "required_status_checks": [
          { "context": "pre-commit", "integration_id": null },
          { "context": "PR conventions", "integration_id": null },
          { "context": "commit messages", "integration_id": null }
        ]
      }
    },
    { "type": "deletion" },
    { "type": "non_fast_forward" }
  ],
  "bypass_actors": []
}
"@

$tmp = [System.IO.Path]::GetTempFileName()
Set-Content -Path $tmp -Value $payload -Encoding UTF8

try {
  gh api --method POST "repos/$Repo/rulesets" --input $tmp | Out-Null
  Write-Host "==> ruleset created via API"
  Write-Host "Done. Direct pushes and force-pushes to $Branch are blocked on GitHub."
  exit 0
} catch {
  Write-Host "==> ruleset API failed; trying classic branch protection…"
}

$classic = @"
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["pre-commit", "PR conventions", "commit messages"]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": false,
    "required_approving_review_count": $Approvals,
    "require_last_push_approval": false
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_conversation_resolution": true
}
"@
Set-Content -Path $tmp -Value $classic -Encoding UTF8

try {
  gh api --method PUT "repos/$Repo/branches/$Branch/protection" `
    -H "Accept: application/vnd.github+json" `
    --input $tmp | Out-Null
  Write-Host "==> classic branch protection enabled"
  Write-Host "Done. Direct pushes and force-pushes to $Branch are blocked on GitHub."
} catch {
  Remove-Item -Force $tmp -ErrorAction SilentlyContinue
  throw "Could not enable protection. Need admin rights on $Repo. Set it manually under Settings → Rules / Branches."
} finally {
  Remove-Item -Force $tmp -ErrorAction SilentlyContinue
}
