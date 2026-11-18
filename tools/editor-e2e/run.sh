#!/usr/bin/env sh
# Browser tests for the level editor. Needs Chrome and Node; no npm packages.
# Runs the editor against scratch copies of both level folders, never the
# repository's.
set -e
cd "$(dirname "$0")"
ROOT=../..
WORK=$(mktemp -d)
mkdir -p "$WORK/builtin" shots
cp "$ROOT"/pkg/level/data/1-1.json "$WORK/builtin/"
python i18n_check.py "$ROOT"
PORT=${PORT:-47399}
start() {
  rm -rf "$WORK/local"
  "$ROOT"/build/super-mario.exe edit --port "$PORT" --dir "$WORK/builtin" --local "$WORK/local" > "$WORK/edit.log" 2>&1 &
  PID=$!
  sleep 2
}
trap 'kill $PID 2>/dev/null; rm -rf "$WORK"' EXIT
export BUILTIN_DIR="$WORK/builtin" LOCAL_DIR="$WORK/local"
start
node e2e.mjs "http://127.0.0.1:$PORT/"
node e2e2.mjs "http://127.0.0.1:$PORT/"
kill $PID; PORT=$((PORT+1)); start
node e2e3.mjs "http://127.0.0.1:$PORT/"
