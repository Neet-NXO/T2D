#!/usr/bin/env bash
set -euo pipefail

DEV="${DEV:?set DEV, for example DEV=eth0}"
ACTION="${ACTION:-apply}"
LOSS="${LOSS:-1%}"
DELAY="${DELAY:-80ms}"
RATE="${RATE:-1gbit}"

case "$ACTION" in
  apply)
    tc qdisc replace dev "$DEV" root netem loss "$LOSS" delay "$DELAY" rate "$RATE"
    ;;
  clear)
    tc qdisc del dev "$DEV" root 2>/dev/null || true
    ;;
  show)
    tc qdisc show dev "$DEV"
    ;;
  *)
    echo "ACTION must be apply, clear, or show" >&2
    exit 2
    ;;
esac
