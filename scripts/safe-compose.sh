#!/usr/bin/env bash
set -Eeuo pipefail

if command -v ramuni-resource-guard >/dev/null 2>&1; then
  ramuni-resource-guard --status
fi

if command -v ramuni-safe-build >/dev/null 2>&1; then
  exec ramuni-safe-build compose -- "$@"
fi

exec docker compose "$@"
