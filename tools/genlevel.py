#!/usr/bin/env python3
"""Generate levels/1-1.json.

The level is laid out here rather than hand-typed so the 212-wide rows stay
aligned. The output is plain declarative JSON: no expressions, no code.
"""
import json
import os

W, H = 212, 15
FLOOR_TOP = 13  # rows 13 and 14 are ground

grid = [[" "] * W for _ in range(H)]
contents = []
entities = []


def put(x, y, ch):
    if 0 <= x < W and 0 <= y < H:
        grid[y][x] = ch


def ground(x0, x1):
    for x in range(x0, x1):
        put(x, FLOOR_TOP, "X")
        put(x, FLOOR_TOP + 1, "X")


def brick(x, y):
    put(x, y, "B")


def question(x, y, kind="coin", count=1, hidden=False):
    if hidden:
        # hidden blocks occupy no visible tile; they are hit as if solid
        put(x, y, " ")
    else:
        put(x, y, "?")
    contents.append({"x": x, "y": y, "kind": kind, "count": count, "hidden": hidden})


def pipe(x, height):
    top = FLOOR_TOP - height
    put(x, top, "[")
    put(x + 1, top, "]")
    for y in range(top + 1, FLOOR_TOP):
        put(x, y, "{")
        put(x + 1, y, "}")


def goomba(x, y=FLOOR_TOP - 1):
    entities.append({"kind": "goomba", "x": x, "y": y})


def koopa(x, y=FLOOR_TOP - 1):
    entities.append({"kind": "koopa", "x": x, "y": y})


# --- ground, with two gaps ------------------------------------------------
GAPS = [(69, 72), (86, 89)]
ground(0, W)
for g0, g1 in GAPS:
    for x in range(g0, g1):
        put(x, FLOOR_TOP, " ")
        put(x, FLOOR_TOP + 1, " ")

# --- opening: one coin block, then the first brick row --------------------
question(16, 9, "coin")

brick(20, 9)
question(21, 9, "mushroom")   # first mushroom: lets the player break bricks
brick(22, 9)
question(23, 9, "coin")
brick(24, 9)
question(22, 5, "coin")       # the high block above the row

# --- four pipes, rising height -------------------------------------------
pipe(28, 2)
pipe(38, 3)
pipe(46, 4)
pipe(57, 4)

# --- brick cluster before the first gap ----------------------------------
brick(64, 9)
brick(65, 9)
question(66, 9, "coin")
brick(67, 9)

# hidden block: an extra life, only reachable by hitting empty air
question(64, 5, "coin", count=1, hidden=True)

# --- after the first gap -------------------------------------------------
# The brick shelf stops well before the second gap: a ceiling directly over
# the take-off point clips the jump and makes the gap impossible.
brick(77, 9)
brick(78, 9)
brick(79, 9)
brick(80, 9)
question(80, 5, "coin")
brick(81, 9)
brick(82, 9)
brick(83, 9)

# --- brick row with a mushroom and a star --------------------------------
brick(94, 9)
question(95, 9, "star")
brick(96, 9)
brick(97, 9)

# --- coin arc over the second gap ----------------------------------------
for i, x in enumerate(range(87, 90)):
    put(x, 8 - (i == 1), "o")

# --- staircase up --------------------------------------------------------
for i in range(4):
    for y in range(FLOOR_TOP - 1 - i, FLOOR_TOP):
        put(134 + i, y, "H")

# --- staircase down ------------------------------------------------------
for i in range(4):
    for y in range(FLOOR_TOP - 4 + i, FLOOR_TOP):
        put(141 + i, y, "H")

# --- final staircase -----------------------------------------------------
for i in range(5):
    for y in range(FLOOR_TOP - 1 - i, FLOOR_TOP):
        put(155 + i, y, "H")

# --- flagpole ------------------------------------------------------------
FLAG_X = 198
put(FLAG_X, 2, "T")
for y in range(3, FLOOR_TOP):
    put(FLAG_X, y, "F")

# --- enemies -------------------------------------------------------------
for x in (22, 40, 51, 52, 80, 97, 98):
    goomba(x)
koopa(110)
koopa(125)

rows = ["".join(r) for r in grid]

level = {
    "name": "1-1",
    "world": "1-1",
    "width": W,
    "height": H,
    "timeLimit": 400,
    "spawn": {"x": 2, "y": FLOOR_TOP - 1},
    "flag": {"x": FLAG_X, "topY": 2, "bottomY": FLOOR_TOP - 1},
    "rows": rows,
    "contents": contents,
    "entities": entities,
}

out_dir = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "levels")
os.makedirs(out_dir, exist_ok=True)
path = os.path.join(out_dir, "1-1.json")
with open(path, "w", encoding="utf-8") as f:
    json.dump(level, f, ensure_ascii=False, indent=1)
print(f"wrote {path} ({W}x{H}, {len(contents)} blocks, {len(entities)} entities)")
