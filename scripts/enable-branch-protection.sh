#!/usr/bin/env bash
# Apply GitHub branch protection / ruleset for main (remote backstop).
# Requires: gh auth login  (admin on the repo)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

if ! command -v gh >/dev/null 2>&1; then
  echo "error: GitHub CLI (gh) is required — https://cli.github.com/" >&2
  exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "error: not logged in — run: gh auth login" >&2
  exit 1
fi

REPO="$(gh repo view --json nameWithOwner -q .nameWithOwner)"
BRANCH="${1:-main}"
APPROVALS="${REQUIRED_APPROVALS:-0}"

echo "==> enabling protection for ${REPO}@${BRANCH}"
echo "    required approvals: ${APPROVALS}"

# Prefer repository rulesets (modern). Fall back to classic branch protection.
payload="$(cat <<EOF
{
  "name": "Protect ${BRANCH}",
  "target": "branch",
  "enforcement": "active",
  "conditions": {
    "ref_name": {
      "include": ["refs/heads/${BRANCH}"],
      "exclude": []
    }
  },
  "rules": [
    {
      "type": "pull_request",
      "parameters": {
        "required_approving_review_count": ${APPROVALS},
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
EOF
)"

tmp="$(mktemp)"
printf '%s' "${payload}" > "${tmp}"

if gh api --method POST "repos/${REPO}/rulesets" --input "${tmp}" >/dev/null; then
  echo "==> ruleset created via API"
  rm -f "${tmp}"
  echo "Done. Direct pushes and force-pushes to ${BRANCH} are blocked on GitHub."
  exit 0
fi

echo "==> ruleset API failed; trying classic branch protection…"

classic="$(cat <<EOF
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["pre-commit", "PR conventions", "commit messages"]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": false,
    "required_approving_review_count": ${APPROVALS},
    "require_last_push_approval": false
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_conversation_resolution": true
}
EOF
)"
printf '%s' "${classic}" > "${tmp}"

if gh api --method PUT "repos/${REPO}/branches/${BRANCH}/protection" \
  -H "Accept: application/vnd.github+json" \
  --input "${tmp}"; then
  rm -f "${tmp}"
  echo "==> classic branch protection enabled"
  echo "Done. Direct pushes and force-pushes to ${BRANCH} are blocked on GitHub."
  exit 0
fi

rm -f "${tmp}"
echo "error: could not enable protection. Need admin rights on ${REPO}." >&2
echo "Manual: GitHub → Settings → Rules → Rulesets (or Branches → Branch protection)" >&2
exit 1
