#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -gt 3 ]; then
  printf "Usage: %s [pid|-] [samples=30] [interval_seconds=10]\n" "$0"
  printf "Tip: use '-' to auto-detect pid with custom sampling values\n"
  exit 1
fi

auto_detect_pid() {
  mapfile -t all_pids < <(pgrep -f "c3lsp" || true)

  candidates=()
  for pid in "${all_pids[@]}"; do
    if [ "$pid" = "$$" ] || [ "$pid" = "$PPID" ]; then
      continue
    fi

    cmd="$(ps -o command= -p "$pid" 2>/dev/null || true)"
    if [[ -z "$cmd" || "$cmd" == *"c3lsp-mem-watch.sh"* ]]; then
      continue
    fi

    candidates+=("$pid")
  done

  if [ "${#candidates[@]}" -eq 0 ]; then
    return 1
  fi

  latest_pid="${candidates[0]}"
  for pid in "${candidates[@]}"; do
    if [ "$pid" -gt "$latest_pid" ]; then
      latest_pid="$pid"
    fi
  done

  printf "%s" "$latest_pid"
}

PID=""
if [ "$#" -eq 0 ] || [ "${1:-}" = "-" ]; then
  PID="$(auto_detect_pid || true)"
  if [ -z "$PID" ]; then
    printf "No running c3lsp process found for auto-detection\n"
    exit 1
  fi
  printf "Auto-detected c3lsp pid: %s\n" "$PID"
else
  if ! [[ "$1" =~ ^[0-9]+$ ]]; then
    printf "pid must be numeric (or use '-' for auto-detect)\n"
    exit 1
  fi
  PID="$1"
fi

SAMPLES="${2:-30}"
INTERVAL="${3:-10}"

if ! [[ "$SAMPLES" =~ ^[0-9]+$ ]] || ! [[ "$INTERVAL" =~ ^[0-9]+$ ]]; then
  printf "samples and interval_seconds must be integers\n"
  exit 1
fi

if ! ps -p "$PID" >/dev/null 2>&1; then
  printf "Process %s not found\n" "$PID"
  exit 1
fi

printf "== Baseline ==\n"
ps -o pid,ppid,%cpu,%mem,rss,vsz,time,command -p "$PID"
printf "\n== vmmap summary ==\n"
vmmap -summary "$PID" | rg "Physical footprint|Physical footprint \(peak\)|TOTAL"

printf "\n== Sampling (%s samples, %ss interval) ==\n" "$SAMPLES" "$INTERVAL"
for ((i = 1; i <= SAMPLES; i++)); do
  date "+%Y-%m-%d %H:%M:%S"
  ps -o pid,%cpu,%mem,rss,vsz,time,state -p "$PID"
  sleep "$INTERVAL"
done

printf "\n== Final vmmap summary ==\n"
vmmap -summary "$PID" | rg "Physical footprint|Physical footprint \(peak\)|TOTAL"
