#!/usr/bin/env sh
# Acceptance check 9.1, end to end. Builds a level in the browser editor, saves
# it, compiles it into the game and plays it through level select.
#
# Everything happens in a throwaway git worktree, so the repository's own
# level directory is never touched.
set -e
cd "$(dirname "$0")/../.."
ROOT=$(pwd)
WT=$(mktemp -d)/wt
git worktree add --detach "$WT" HEAD >/dev/null
trap 'git -C "$ROOT" worktree remove --force "$WT" >/dev/null 2>&1; kill $PID 2>/dev/null' EXIT
# Bring the working tree's uncommitted changes along.
git diff HEAD | git -C "$WT" apply --whitespace=nowarn
git ls-files --others --exclude-standard -z | xargs -0 -I{} sh -c 'mkdir -p "$(dirname "$1/$2")"; cp "$2" "$1/$2"' _ "$WT" {}
# Level files are LF by .gitattributes. Until that file is committed, a
# worktree from HEAD may have been checked out with CRLF; put them back.
for f in "$WT"/pkg/level/data/*.json; do tr -d '\r' < "$f" > "$f.lf" && mv "$f.lf" "$f"; done

cd "$WT"
make build >/dev/null
PORT=${PORT:-47398}
./build/super-mario.exe edit --no-browser --port "$PORT" --dir pkg/level/data --draft-dir build/accept-drafts > edit.log 2>&1 &
PID=$!
sleep 2
mkdir -p tools/editor-e2e/shots
(cd tools/editor-e2e && node accept.mjs "http://127.0.0.1:$PORT/")
kill $PID 2>/dev/null || true

echo "--- saved level ---"
head -12 pkg/level/data/c-tester-snow-path.json
echo "--- repository checks with the new level ---"
go test ./pkg/level/...
echo "--- play it through level select ---"
make build >/dev/null
ACCEPT_LEVEL=c-tester-snow-path go test -run TestAcceptanceLevelPlaysThroughSelect -v ./pkg/app 2>&1 | grep -E "^(---|ok|FAIL)|cleared|list:|雪原|1-1"
GOOS=js GOARCH=wasm go build -o /dev/null ./cmd/wasm && echo "wasm build ok"
