#!/usr/bin/env python3
"""Generate the built-in levels 1-2 to 2-4 into pkg/level/data/.

Together with 1-1 (tools/genlevel.py) they make eight levels, one per theme,
each built around a different way of playing:

  1-1 grassland    the basics (genlevel.py)
  1-2 underground  a cave of brick ceilings: a low corridor, an upper route
                   along the top of the ceiling, brick walls to jump or smash
  1-3 dawn         a sky course: no floor, only pillars and floating ledges
  1-4 lava         a castle: lava pits under a low ceiling of hanging rock,
                   and a brick bridge guarded by two koopas
  2-1 snow         climbing: towers, a brick tower and tall staircases
  2-2 desert       shells: koopas lined up in front of rows of goombas, and
                   pipes for the shells to bounce between
  2-3 dusk         a long bridge in the evening sky, on a short clock
  2-4 night        a gauntlet lit by star blocks, with hidden blocks to find

Every level is checked by the repository tests: it must load, be in the
canonical layout, and raise no design warnings (the reach check works out from
the game's own jump physics that the flag can be reached from the start).
Run `make gen-level` after editing; it also puts the files in the canonical
layout.
"""
import json
import os

H = 15
FLOOR = 13  # the top row of the ground; a player on it stands in row 12
OUT = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "pkg", "level", "data")


class Level:
    def __init__(self, name, width, title, description, difficulty, theme, time_limit=400):
        self.name, self.w = name, width
        self.meta = {"title": title, "author": "super-mario", "description": description, "difficulty": difficulty}
        self.style = {"theme": theme}
        self.time = time_limit
        self.grid = [[" "] * width for _ in range(H)]
        self.contents, self.entities = [], []
        self.spawn = (2, FLOOR - 1)
        self.flag = None

    # --- terrain ---------------------------------------------------------
    def put(self, x, y, ch):
        if 0 <= x < self.w and 0 <= y < H:
            self.grid[y][x] = ch

    def fill(self, x0, x1, y0, y1, ch):
        """Fill columns x0..x1-1, rows y0..y1 inclusive."""
        for x in range(x0, x1):
            for y in range(y0, y1 + 1):
                self.put(x, y, ch)

    def ground(self, x0, x1, top=FLOOR):
        self.fill(x0, x1, top, H - 1, "X")

    def pit(self, x0, x1):
        self.fill(x0, x1, 0, H - 1, " ")

    def column(self, x0, x1, top, ch="H"):
        """A pillar from row top down to the bottom of the level."""
        self.fill(x0, x1, top, H - 1, ch)

    def ledge(self, x0, x1, y, ch="H"):
        """A one-row platform in the air."""
        self.fill(x0, x1, y, y, ch)

    def stair_up(self, x, steps, base=FLOOR):
        for i in range(steps):
            self.fill(x + i, x + i + 1, base - 1 - i, base - 1, "H")

    def stair_down(self, x, steps, base=FLOOR):
        for i in range(steps):
            self.fill(x + i, x + i + 1, base - steps + i, base - 1, "H")

    def pipe(self, x, height, floor=FLOOR):
        top = floor - height
        self.put(x, top, "[")
        self.put(x + 1, top, "]")
        for y in range(top + 1, floor):
            self.put(x, y, "{")
            self.put(x + 1, y, "}")

    def coins(self, x0, x1, y, step=1):
        for x in range(x0, x1, step):
            self.put(x, y, "o")

    def arc(self, x0, x1, y):
        """Coins in a low arc over a gap, one row higher in the middle."""
        mid = (x0 + x1 - 1) / 2
        for x in range(x0, x1):
            self.put(x, y - (1 if abs(x - mid) < (x1 - x0) / 4 else 0), "o")

    # --- blocks ------------------------------------------------------------
    def question(self, x, y, kind="coin", count=1):
        self.put(x, y, "?")
        self.contents.append({"x": x, "y": y, "kind": kind, "count": count})

    def brick_with(self, x, y, kind="coin", count=1):
        """A brick that holds something: a coin brick when count > 1."""
        self.put(x, y, "B")
        self.contents.append({"x": x, "y": y, "kind": kind, "count": count})

    def hidden(self, x, y, kind="coin"):
        self.put(x, y, " ")
        self.contents.append({"x": x, "y": y, "kind": kind, "hidden": True})

    # --- enemies -----------------------------------------------------------
    def _stand(self, x):
        """The row an enemy at column x stands in: on the first solid cell
        below open air, so a roof over the level does not count."""
        solid = "XHB?[]{}"
        y = 0
        while y < H and self.grid[y][x] in solid:
            y += 1  # through the roof
        for yy in range(y, H):
            if self.grid[yy][x] in solid:
                return yy - 1
        raise ValueError(f"{self.name}: nothing to stand on at column {x}")

    def goomba(self, *xs):
        for x in xs:
            self.entities.append({"kind": "goomba", "x": x, "y": self._stand(x)})

    def koopa(self, *xs):
        for x in xs:
            self.entities.append({"kind": "koopa", "x": x, "y": self._stand(x)})

    # --- the end -------------------------------------------------------------
    def finish(self, flag_x, bottom=FLOOR - 1):
        """The flagpole, ten rows tall, standing on the ground at flag_x."""
        top = bottom - 10
        self.put(flag_x, top, "T")
        for y in range(top + 1, bottom + 1):
            self.put(flag_x, y, "F")
        self.flag = {"x": flag_x, "topY": top, "bottomY": bottom}

    def write(self):
        assert self.flag, f"{self.name}: no flagpole"
        level = {
            "name": self.name,
            "world": self.name,
            "width": self.w,
            "height": H,
            "timeLimit": self.time,
            "meta": self.meta,
            "style": self.style,
            "spawn": {"x": self.spawn[0], "y": self.spawn[1]},
            "flag": self.flag,
            "rows": ["".join(r) for r in self.grid],
            "contents": self.contents,
            "entities": self.entities,
        }
        os.makedirs(OUT, exist_ok=True)
        path = os.path.join(OUT, self.name + ".json")
        with open(path, "w", encoding="utf-8", newline="\n") as f:
            json.dump(level, f, ensure_ascii=False, indent=1)
        print(f"wrote {path} ({self.w} wide, {len(self.contents)} blocks, {len(self.entities)} enemies)")


# =============================================================================
# 1-2 underground: brick ceilings
# =============================================================================
def level_1_2():
    L = Level("1-2", 168, "地下砖窟", "低矮的砖块走廊里踢龟壳，砖顶上还有一条藏着金币的近道。", 2, "underground")
    L.ground(0, L.w)
    for a, b in [(58, 61), (94, 98), (126, 129)]:
        L.pit(a, b)
    L.fill(0, 147, 0, 1, "X")  # the cave roof

    # A brick row with the first mushroom and a coin brick.
    L.fill(12, 20, 9, 9, "B")
    L.question(14, 9, "mushroom")
    L.brick_with(18, 9, "coin", 6)
    L.goomba(24, 26)

    # Steps up to the top of the corridor's ceiling: the upper route.
    L.stair_up(28, 4)
    # The corridor: a brick ceiling three rows over the floor. Coins along the
    # floor inside, more along the top; a koopa to kick down the corridor.
    L.fill(34, 57, 8, 8, "B")
    L.question(40, 8, "coin")
    L.question(48, 8, "star")
    L.coins(36, 55, 11, 2)
    L.coins(36, 55, 5, 2)
    L.koopa(52)

    # Brick walls: jump them, or smash a way through when big.
    L.fill(70, 71, 8, 12, "B")
    L.pipe(76, 3)
    L.fill(84, 86, 8, 12, "B")
    L.question(85, 4, "coin")
    L.goomba(73, 80, 89, 91)

    # Pipes with goombas between them, and a hidden extra life.
    L.pipe(102, 2)
    L.pipe(108, 4)
    L.pipe(116, 3)
    L.hidden(112, 9, "oneup")
    L.goomba(105, 113, 114, 120)

    L.stair_up(136, 5)
    L.finish(148)
    L.write()


# =============================================================================
# 1-3 dawn: a sky course
# =============================================================================
def level_1_3():
    L = Level("1-3", 168, "黎明云阶", "没有地面，只有石柱和悬空的平台，一步一步跳过去。", 3, "dawn")
    L.ground(0, 14)
    # Pillars run down to the bottom of the level; ledges hang in the air.
    L.column(18, 21, 11)
    L.column(25, 28, 9)
    L.ledge(32, 36, 8)
    L.column(40, 43, 11)
    L.column(46, 54, 10)
    L.koopa(52)
    L.column(58, 60, 8)
    L.column(64, 70, 10)
    L.goomba(68)
    L.ledge(74, 79, 7, "B")
    L.question(76, 3, "mushroom")
    L.column(83, 86, 12)
    L.column(90, 94, 10)
    L.ledge(98, 101, 9)
    L.column(105, 114, 11)  # a safe place to stop before the final climb
    L.column(116, 118, 8)
    L.ledge(122, 125, 6)
    L.coins(122, 125, 4)
    L.column(129, 134, 9)
    L.koopa(133)
    L.column(138, 141, 11)
    L.ground(145, L.w)
    # Coins over some of the gaps show the way.
    for a, b, y in [(15, 18, 9), (29, 32, 6), (55, 58, 6), (80, 83, 5), (102, 105, 7), (119, 122, 5)]:
        L.arc(a, b, y)
    for x, y in [(18, 10), (32, 7), (58, 7), (83, 11), (105, 10), (122, 5)]:
        L.put(x, y, "o")
    L.stair_up(148, 4)
    L.finish(158)
    L.write()


# =============================================================================
# 1-4 lava: the castle
# =============================================================================
def level_1_4():
    L = Level("1-4", 176, "熔岩城堡", "先练短跳，再助跑穿过低矮岩顶；终点前有两只乌龟守着错层砖桥。", 4, "lava", 300)
    segments = [(0, 20), (23, 34), (38, 52), (55, 61), (63, 80), (84, 96), (101, 118), (121, 132), (152, 176)]
    for a, b in segments:
        L.ground(a, b)
    # Three bridge sections with a raised centre and room to pause on each.
    for a, b, row in [(132, 138, 13), (140, 146, 12), (148, 152, 13)]:
        L.ledge(a, b, row, "B")
    L.fill(0, 156, 0, 2, "H")  # the castle roof

    # Rock hanging from the roof: jumps under it have to stay low.
    for a, b, bottom in [(27, 30, 7), (44, 47, 8), (50, 55, 7), (66, 69, 8), (88, 91, 7), (104, 108, 8), (118, 121, 6)]:
        L.fill(a, b, 3, bottom, "H")

    L.question(10, 9, "mushroom")
    L.question(15, 9, "coin")
    # Raised steps inside the castle.
    L.fill(40, 43, 11, 12, "H")
    L.fill(72, 75, 10, 12, "H")
    L.fill(110, 113, 10, 12, "H")
    L.question(73, 6, "coin")

    for x, y in [(19, 11), (21, 10), (23, 11), (51, 10), (53, 10), (55, 11), (138, 10), (140, 10), (147, 10)]:
        L.put(x, y, "o")
    L.goomba(31, 46, 76, 78, 89, 92, 103, 115)
    L.koopa(125)
    L.koopa(135, 143)  # on the first and raised bridge sections
    L.finish(162)
    L.write()


# =============================================================================
# 2-1 snow: climbing
# =============================================================================
def level_2_1():
    L = Level("2-1", 168, "雪原高塔", "一路往上爬：石柱、砖塔和高高的台阶，乌龟从上面走下来。", 3, "snow")
    L.ground(0, L.w)
    for a, b in [(40, 43), (80, 84), (120, 124)]:
        L.pit(a, b)

    L.stair_up(12, 4)
    L.column(21, 23, 10)
    L.column(25, 28, 7)
    L.question(26, 3, "mushroom")
    L.koopa(27)
    L.stair_up(33, 6)

    # A brick tower too tall to jump: climb the step beside it.
    L.column(47, 49, 10)
    L.fill(51, 53, 6, 12, "B")
    L.question(52, 2, "oneup")
    L.goomba(57, 60)

    # Pillars of different heights.
    L.column(63, 65, 9)
    L.column(68, 70, 7)
    L.column(73, 75, 9)
    L.column(77, 79, 8)
    L.koopa(69)

    # Brick walls with goombas between.
    L.fill(92, 93, 9, 12, "B")
    L.fill(100, 102, 7, 12, "B")
    L.fill(110, 111, 8, 12, "B")
    L.question(101, 3, "coin")
    L.koopa(96)
    L.goomba(105, 107, 114, 116)

    L.stair_up(130, 6)
    L.finish(146)
    L.write()


# =============================================================================
# 2-2 desert: shells
# =============================================================================
def level_2_2():
    L = Level("2-2", 190, "沙漠龟壳", "乌龟排在成群的蘑菇怪前面：踩出壳一脚踢过去，看它在水管之间来回撞。", 2, "desert")
    L.ground(0, L.w)
    for a, b in [(88, 90), (142, 144)]:
        L.pit(a, b)

    L.question(12, 9, "coin")
    L.koopa(20)
    L.goomba(26, 27, 28, 29)

    # Two pipes for a shell to bounce between.
    L.pipe(40, 2)
    L.pipe(54, 2)
    L.koopa(43)
    L.goomba(46, 48, 50)

    L.koopa(64)
    L.goomba(70, 72, 74, 76)
    L.column(80, 87, 10)
    L.goomba(82, 85)

    L.koopa(96)
    L.goomba(100, 102, 104, 106)
    L.pipe(112, 3)

    # A star, then a crowd to run through while it lasts.
    L.fill(116, 123, 9, 9, "B")
    L.question(119, 9, "star")
    L.question(121, 5, "coin")
    L.goomba(126, 128, 130, 132, 134, 136, 138)

    L.koopa(150, 154)
    L.stair_up(160, 5)
    L.finish(172)
    L.write()


# =============================================================================
# 2-3 dusk: the long bridge
# =============================================================================
def level_2_3():
    L = Level("2-3", 200, "黄昏长桥", "在晚霞里跑过一座断断续续的长桥，时间很紧，别停下来。", 3, "dusk", 100)
    L.ground(0, 15)
    # The bridge: bricks in row 10, standing in row 9, with nothing under it.
    BRIDGE = 10
    spans = [(17, 38), (42, 58), (62, 64), (68, 85), (90, 104), (108, 111), (115, 135), (140, 158)]
    for a, b in spans:
        L.ledge(a, b, BRIDGE, "B")
    L.ground(163, L.w)
    # The opening jump climbs three rows from the ground onto the bridge.
    for x, y in [(14, 11), (15, 9), (16, 7), (17, 7), (19, 8)]:
        L.put(x, y, "o")

    # Coins over every gap.
    prev = 15
    for a, b in spans:
        if a - prev >= 3:
            L.arc(prev, a, BRIDGE - 4)
        prev = b
    L.arc(158, 163, BRIDGE - 4)

    for x in (24, 50, 76, 122, 146):
        L.question(x, BRIDGE - 5, "coin")
    L.question(96, BRIDGE - 5, "mushroom")

    L.goomba(30, 34, 52, 78, 81, 125, 130, 150)
    L.koopa(98)

    L.stair_up(170, 5)
    L.finish(182)
    L.write()


# =============================================================================
# 2-4 night: the star gauntlet
# =============================================================================
def level_2_4():
    L = Level("2-4", 176, "星夜突围", "夜里敌人成群结队，抢到星星一路冲过去；暗处还藏着看不见的方块。", 4, "night")
    L.ground(0, L.w)
    for a, b in [(100, 103), (134, 137)]:
        L.pit(a, b)

    L.question(12, 9, "star")
    L.question(14, 9, "mushroom")
    L.goomba(22, 24, 26, 28, 30, 32, 34, 36, 38, 40)

    # An invisible staircase of hidden coin blocks, rising to the right.
    L.hidden(50, 9, "coin")
    L.hidden(53, 7, "coin")
    L.hidden(56, 5, "oneup")

    L.fill(62, 64, 9, 12, "B")
    L.question(63, 5, "coin")
    L.pipe(70, 3)
    L.question(76, 9, "star")
    L.koopa(80, 84)
    L.goomba(86, 88, 90, 92, 94, 96)

    L.question(108, 9, "star")
    L.goomba(114, 116, 118, 120, 122, 124, 126, 128, 130)

    L.stair_up(146, 5)
    L.finish(158)
    L.write()


if __name__ == "__main__":
    for build in (level_1_2, level_1_3, level_1_4, level_2_1, level_2_2, level_2_3, level_2_4):
        build()
