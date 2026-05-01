#!/usr/bin/env bash
# Seed labels for JamesPagetButler/confluent-trust.
# Idempotent: re-running updates colors/descriptions on existing labels.
set -euo pipefail

REPO="${REPO:-JamesPagetButler/confluent-trust}"

upsert() {
  local name="$1" color="$2" desc="$3"
  if gh label list -R "$REPO" --search "$name" --json name -q '.[].name' | grep -Fxq "$name"; then
    gh label edit "$name" -R "$REPO" --color "$color" --description "$desc"
  else
    gh label create "$name" -R "$REPO" --color "$color" --description "$desc"
  fi
}

# Phase
upsert "phase:crawl"   "0E8A16" "Crawl phase — JSON storage, core engine"
upsert "phase:walk"    "1D76DB" "Walk phase — MuninnDB integration"
upsert "phase:run"     "5319E7" "Run phase — BMA integration, SurrealDB"

# Type
upsert "type:model"    "D4C5F9" "Data model / types"
upsert "type:compute"  "FBCA04" "Computation / algorithms"
upsert "type:store"    "F9D0C4" "Storage layer"
upsert "type:test"     "BFD4F2" "Tests and fixtures"
upsert "type:infra"    "C2E0C6" "Repo setup, CI, tooling"
upsert "type:doc"      "D3D3D3" "Documentation"

# Priority
upsert "priority:p0"   "B60205" "Must have for phase completion"
upsert "priority:p1"   "D93F0B" "Should have"
upsert "priority:p2"   "E99695" "Nice to have"

# Status (BMA convention)
upsert "status:triage"      "D876E3" "Newly opened, awaiting acceptance"
upsert "status:accepted"    "C5DEF5" "Approved for work"
upsert "status:in-progress" "1D76DB" "Actively being worked on"
upsert "status:blocked"     "B60205" "Blocked by another issue"
upsert "status:done"        "0E8A16" "Acceptance criteria met"

# Misc
upsert "blocked"      "000000" "Blocked by another issue"

echo "labels seeded for $REPO"
