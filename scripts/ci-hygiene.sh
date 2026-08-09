#!/usr/bin/env bash
# CI / local hygiene: fail on tracked local artifacts, secret-like content, and stale module paths.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "ci-hygiene: not a git repository" >&2
  exit 1
fi

failed=0

fail() {
  echo "ci-hygiene: FAIL: $*" >&2
  failed=1
}

echo "ci-hygiene: checking tracked paths..."

while IFS= read -r path; do
  [[ -z "$path" ]] && continue

  if [[ "$path" == ".cursor" || "$path" == .cursor/* ]]; then
    fail "tracked Cursor artifact: $path"
  fi
  if [[ "$path" == ".agents" || "$path" == .agents/* ]]; then
    fail "tracked agents artifact: $path"
  fi
  if [[ "$path" == "AGENTS.md" ]]; then
    fail "tracked AGENTS.md (not for the public tree): $path"
  fi
  if [[ "$path" == *.local.md ]]; then
    fail "tracked local markdown: $path"
  fi
  if [[ "$path" == *.plan.md ]]; then
    fail "tracked plan file: $path"
  fi
  if [[ "$path" == "bin" || "$path" == bin/* ]]; then
    fail "tracked build output under bin/: $path"
  fi
  base="${path##*/}"
  if [[ "$base" == "credentials" ]]; then
    fail "tracked credentials file: $path"
  fi
  if [[ "$path" == "config.yaml" ]]; then
    fail "tracked root config.yaml (site-specific; use examples only): $path"
  fi
done < <(git ls-files)

echo "ci-hygiene: scanning tracked file contents..."

# Allowlist this script: it necessarily mentions some of the forbidden shapes as search needles.
is_allowlisted() {
  case "$1" in
    scripts/ci-hygiene.sh) return 0 ;;
    *) return 1 ;;
  esac
}

report_matches() {
  local label="$1"
  local matches="$2"
  local line file
  [[ -z "$matches" ]] && return 0
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    file="${line%%:*}"
    if is_allowlisted "$file"; then
      continue
    fi
    fail "$label match in $line"
  done <<<"$matches"
}

# Patterns assembled so literal private operator domains are never hardcoded.
# Private IPv4 (RFC1918): 192.168/16, 10/8, 172.16/12
pat_priv_a='192[.]168[.][0-9]{1,3}[.][0-9]{1,3}'
pat_priv_b='(^|[^0-9])10[.][0-9]{1,3}[.][0-9]{1,3}[.][0-9]{1,3}([^0-9]|$)'
pat_priv_c='172[.](1[6-9]|2[0-9]|3[01])[.][0-9]{1,3}[.][0-9]{1,3}'

# Token / key shapes: ghp_, github_pat_, AKIA…, BEGIN … PRIVATE KEY
pat_ghp='ghp_[A-Za-z0-9]{20,}'
pat_gpat='github_pat_[A-Za-z0-9_]{20,}'
pat_akia='AKIA[0-9A-Z]{16}'
pat_pkey='BEGIN (RSA |OPENSSH |EC )?PRIVATE KEY'

scan_re() {
  local label="$1"
  local pattern="$2"
  local matches
  matches="$(git grep -nE -I -e "$pattern" -- . 2>/dev/null || true)"
  report_matches "$label" "$matches"
}

scan_re "private IPv4 (192.168/16)" "$pat_priv_a"
scan_re "private IPv4 (10/8)" "$pat_priv_b"
scan_re "private IPv4 (172.16/12)" "$pat_priv_c"
scan_re "GitHub token (ghp_)" "$pat_ghp"
scan_re "GitHub fine-grained PAT" "$pat_gpat"
scan_re "AWS access key id (AKIA…)" "$pat_akia"
scan_re "private key block" "$pat_pkey"

echo "ci-hygiene: checking for stale module / org remnants..."

# Former org/module path must not remain (canonical module is github.com/knieberg/ldash).
stale_a='ldash-sh/ldash'
stale_b='github.com/ldash-sh/'

report_matches "stale path remnant (ldash-sh/ldash)" \
  "$(git grep -nF -I -e "$stale_a" -- . 2>/dev/null || true)"
report_matches "stale path remnant (github.com/ldash-sh/)" \
  "$(git grep -nF -I -e "$stale_b" -- . 2>/dev/null || true)"

if [[ "$failed" -ne 0 ]]; then
  echo "ci-hygiene: failed" >&2
  exit 1
fi

echo "ci-hygiene: OK"
exit 0
