#!/usr/bin/env bash
set -euo pipefail

# High-confidence scan for committed secrets. This intentionally checks only
# tracked text files so ignored local files such as .env.local are not printed.
checks=(
  "AWS access key::A[SK]IA[0-9A-Z]{16}"
  "Google API key::AIza[0-9A-Za-z_-]{35}"
  "GitHub token::gh[pousr]_[0-9A-Za-z_]{20,}"
  "GitHub fine-grained token::github_pat_[0-9A-Za-z_]{20,}"
  "OpenAI-style API key::sk-[0-9A-Za-z_-]{20,}"
  "Slack token::xox[baprs]-[0-9A-Za-z-]{20,}"
  "JWT::eyJ[0-9A-Za-z_-]{10,}\.[0-9A-Za-z_-]{10,}\.[0-9A-Za-z_-]{10,}"
  "Private key::-----BEGIN (RSA |DSA |EC |OPENSSH |PGP )?PRIVATE KEY-----"
)

failed=0

for check in "${checks[@]}"; do
  name="${check%%::*}"
  pattern="${check#*::}"
  matches="$(
    git grep -IEn "$pattern" -- \
      ':!frontend/package-lock.json' \
      ':!go.sum' \
      ':!frontend/dist/**' \
      2>/dev/null || true
  )"
  if [[ -n "$matches" ]]; then
    echo "::error title=Potential secret::${name} pattern matched"
    echo "$matches" | awk -F: '{print "  " $1 ":" $2}' | sort -u
    failed=1
  fi
done

if [[ "$failed" -ne 0 ]]; then
  echo "Potential committed secrets found. Review the file/line locations above."
  exit 1
fi

echo "No high-confidence committed secrets found."
