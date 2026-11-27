// model.js — the level as the editor holds it, and every edit on it.
//
// Nothing here touches the page, so all of it runs (and is tested) in Node.
// The level is a structured document rather than fifteen strings: tools,
// undo and the clipboard all work on cells, and only saving turns it back
// into the file format (design decision 3). The file's canonical layout is
// produced by the game's own formatter in WebAssembly, never here.

export const HEIGHT = 15;

// Tile glyphs, exactly as level files write them.
export const T = {
  AIR: ' ', GROUND: 'X', STAIR: 'H', BRICK: 'B', QUESTION: '?',
  PIPE_LIP_L: '[', PIPE_LIP_R: ']', PIPE_BODY_L: '{', PIPE_BODY_R: '}',
  POLE: 'F', BALL: 'T', COIN: 'o',
};

export const BLOCK_KINDS = ['coin', 'mushroom', 'star', 'oneup', 'poison', 'gold'];

// ---------------------------------------------------------------- document

/** A blank level of a width, all air. */
export function blank(width) {
  return {
    name: '', world: '', width, timeLimit: 400,
    meta: {}, style: {},
    tiles: new Array(width * HEIGHT).fill(T.AIR),
    // contents: "x,y" -> {kind, count, hidden, pool}
    contents: new Map(),
    // entities: [{kind, x, y}]
    entities: [],
    spawn: { x: 2, y: 12 },
    flag: { x: width - 3, topY: 2, bottomY: 12 },
  };
}

/** The level a "new level" starts from: ground, spawn and flag, ready to play. */
export function starter(width = 120, theme = 'grassland') {
  const lv = blank(width);
  lv.style = { theme };
  fillGround(lv, { layers: 2, keepGaps: false });
  lv.flag = { x: width - 8, topY: 2, bottomY: 12 };
  placeFlagTiles(lv);
  return lv;
}

export const key = (x, y) => x + ',' + y;
export const at = (lv, x, y) => lv.tiles[y * lv.width + x];
export const inside = (lv, x, y) => x >= 0 && x < lv.width && y >= 0 && y < HEIGHT;

/** Parse the game's level file text into a document. */
export function fromFile(text) {
  const f = JSON.parse(text);
  if (f.height !== HEIGHT || f.rows?.length !== HEIGHT) {
    throw new RangeError('unsupported level height');
  }
  const lv = blank(f.width);
  lv.name = f.name || ''; lv.world = f.world || '';
  lv.timeLimit = f.timeLimit || 400;
  lv.meta = { ...(f.meta || {}) };
  lv.style = structuredClone(f.style || {});
  f.rows.forEach((row, y) => {
    [...row].forEach((ch, x) => { lv.tiles[y * lv.width + x] = ch; });
  });
  for (const c of f.contents || []) {
    lv.contents.set(key(c.x, c.y), {
      kind: c.kind, count: c.count || 1, hidden: !!c.hidden,
      pool: c.pool ? c.pool.map(p => ({ kind: p.kind, w: p.w })) : null,
    });
  }
  lv.entities = (f.entities || []).map(e => ({ kind: e.kind, x: e.x, y: e.y }));
  lv.spawn = { ...f.spawn };
  lv.flag = { ...f.flag };
  return lv;
}

/** Turn the document back into level file JSON (unformatted; WASM formats). */
export function toFile(lv) {
  const rows = [];
  for (let y = 0; y < HEIGHT; y++) {
    rows.push(lv.tiles.slice(y * lv.width, (y + 1) * lv.width).join(''));
  }
  const contents = [];
  for (const [k, c] of lv.contents) {
    const [x, y] = k.split(',').map(Number);
    const out = { x, y, kind: c.kind };
    if (c.count > 1) out.count = c.count;
    if (c.hidden) out.hidden = true;
    if (c.kind === 'random' && c.pool) out.pool = c.pool.map(p => ({ kind: p.kind, w: p.w }));
    contents.push(out);
  }
  const f = {
    name: lv.name || 'untitled', world: lv.world || lv.name || 'untitled',
    width: lv.width, height: HEIGHT, timeLimit: lv.timeLimit,
  };
  const meta = cleanObject(lv.meta);
  if (meta) f.meta = meta;
  const style = cleanObject(lv.style);
  if (style) f.style = style;
  Object.assign(f, { spawn: lv.spawn, flag: lv.flag, rows, contents, entities: lv.entities });
  return JSON.stringify(f);
}

function cleanObject(o) {
  if (!o) return null;
  const out = {};
  for (const [k, v] of Object.entries(o)) {
    if (v !== undefined && v !== null && v !== '' && v !== 0) out[k] = v;
  }
  return Object.keys(out).length ? out : null;
}

// ---------------------------------------------------------------- patches
//
// Every edit is a patch: what it changed, before and after. Undo applies the
// before side, redo the after side. A drag of the pen collects into one patch
// from pointerdown to pointerup, so it undoes as one step (design decision 4).

export function newPatch(label) {
  return { label, cells: new Map(), contents: new Map(), entities: null, spawn: null, flag: null, meta: null, style: null, timeLimit: null, width: null };
}

export function isEmptyPatch(p) {
  return p.cells.size === 0 && p.contents.size === 0 && !p.entities && !p.spawn &&
    !p.flag && !p.meta && !p.style && !p.timeLimit && !p.width;
}

/** Set a tile, recording it in the patch. */
export function setTile(lv, p, x, y, ch) {
  if (!inside(lv, x, y)) return;
  const i = y * lv.width + x;
  if (lv.tiles[i] === ch) return;
  if (!p.cells.has(i)) p.cells.set(i, [lv.tiles[i], ch]);
  else p.cells.get(i)[1] = ch;
  lv.tiles[i] = ch;
}

/** Set (or with null, remove) a block's content, recording it in the patch. */
export function setContent(lv, p, x, y, c) {
  const k = key(x, y);
  const before = lv.contents.has(k) ? structuredClone(lv.contents.get(k)) : null;
  const after = c ? structuredClone(c) : null;
  if (!p.contents.has(k)) p.contents.set(k, [before, after]);
  else p.contents.get(k)[1] = after;
  if (after) lv.contents.set(k, after); else lv.contents.delete(k);
}

function snapEntities(lv, p) {
  if (!p.entities) p.entities = [structuredClone(lv.entities), null];
}
function finishEntities(lv, p) {
  if (p.entities) p.entities[1] = structuredClone(lv.entities);
}

function snapField(lv, p, field) {
  if (!p[field]) p[field] = [structuredClone(lv[field]), null];
}
function finishField(lv, p, field) {
  if (p[field]) p[field][1] = structuredClone(lv[field]);
}

/** Run fn as one recorded change of a whole field (spawn, flag, meta, style). */
export function changeField(lv, p, field, fn) {
  snapField(lv, p, field);
  fn(lv[field]);
  finishField(lv, p, field);
}

/** Apply one side of a patch: 0 = before (undo), 1 = after (redo). */
export function applyPatch(lv, p, side) {
  if (p.width) resizeRaw(lv, p.width[side]);
  for (const [i, pair] of p.cells) lv.tiles[i] = pair[side];
  for (const [k, pair] of p.contents) {
    if (pair[side]) lv.contents.set(k, structuredClone(pair[side])); else lv.contents.delete(k);
  }
  if (p.entities) lv.entities = structuredClone(p.entities[side]);
  for (const f of ['spawn', 'flag', 'meta', 'style', 'timeLimit']) {
    if (p[f]) lv[f] = structuredClone(p[f][side]);
  }
}

/** Set a plain value field (the time limit) as a recorded change. */
export function setValue(lv, p, field, value) {
  if (lv[field] === value) return;
  if (!p[field]) p[field] = [lv[field], value]; else p[field][1] = value;
  lv[field] = value;
}

/** Undo history: at least 200 steps (spec level-editor / 撤销与重做). */
export class History {
  constructor(limit = 500) { this.done = []; this.undone = []; this.limit = limit; }
  push(p) {
    if (isEmptyPatch(p)) return false;
    this.done.push(p);
    if (this.done.length > this.limit) this.done.shift();
    this.undone = [];
    return true;
  }
  undo(lv) {
    const p = this.done.pop(); if (!p) return null;
    applyPatch(lv, p, 0); this.undone.push(p); return p;
  }
  redo(lv) {
    const p = this.undone.pop(); if (!p) return null;
    applyPatch(lv, p, 1); this.done.push(p); return p;
  }
  get canUndo() { return this.done.length > 0; }
  get canRedo() { return this.undone.length > 0; }
}

// ---------------------------------------------------------------- elements
//
// What the palette offers. "Element" is a thing a designer places; most are a
// single tile, some (pipes) are stamps.

export const ELEMENTS = [
  { id: 'ground', label: '地面', glyph: T.GROUND, sprite: 'ground', key: '1' },
  { id: 'brick', label: '砖块', glyph: T.BRICK, sprite: 'brick', key: '2' },
  { id: 'question', label: '问号块', glyph: T.QUESTION, sprite: 'question', key: '3' },
  { id: 'stair', label: '硬块', glyph: T.STAIR, sprite: 'stair', key: '4' },
  { id: 'coin', label: '金币', glyph: T.COIN, sprite: 'coin', key: '5' },
  { id: 'goomba', label: '蘑菇怪', entity: 'goomba', sprite: 'goomba', key: '6' },
  { id: 'koopa', label: '乌龟', entity: 'koopa', sprite: 'koopa', key: '7' },
  { id: 'hidden', label: '隐藏块', hidden: true, sprite: 'question', key: '8' },
  { id: 'spawn', label: '出生点', spawn: true, key: '9' },
  { id: 'flag', label: '旗杆', flag: true, key: '0' },
];

export const elementById = id => ELEMENTS.find(e => e.id === id);

/** Is (x, y) part of the flagpole's own tiles? */
const isFlagGlyph = ch => ch === T.POLE || ch === T.BALL;

/** Remove whatever stands at a cell: tile, block content, enemy. */
export function eraseCell(lv, p, x, y) {
  if (!inside(lv, x, y)) return;
  if (isFlagGlyph(at(lv, x, y))) return; // the flag moves, it is not erased
  setTile(lv, p, x, y, T.AIR);
  if (lv.contents.has(key(x, y))) setContent(lv, p, x, y, null);
  const i = lv.entities.findIndex(e => e.x === x && e.y === y);
  if (i >= 0) {
    snapEntities(lv, p);
    lv.entities.splice(i, 1);
    finishEntities(lv, p);
  }
}

/** Place an element at a cell. `content` is the block content to give a
 * question or hidden block (the palette's current choice). */
export function placeElement(lv, p, el, x, y, content) {
  if (!inside(lv, x, y)) return;
  if (el.spawn) {
    changeField(lv, p, 'spawn', s => { s.x = x; s.y = y; });
    return;
  }
  if (el.flag) {
    moveFlag(lv, p, x);
    return;
  }
  if (isFlagGlyph(at(lv, x, y))) return;
  if (el.entity) {
    snapEntities(lv, p);
    lv.entities = lv.entities.filter(e => !(e.x === x && e.y === y));
    lv.entities.push({ kind: el.entity, x, y });
    finishEntities(lv, p);
    if (at(lv, x, y) !== T.AIR) setTile(lv, p, x, y, T.AIR);
    return;
  }
  if (el.hidden) {
    setTile(lv, p, x, y, T.AIR);
    setContent(lv, p, x, y, { ...(content || defaultContent()), hidden: true });
    return;
  }
  setTile(lv, p, x, y, el.glyph);
  if (el.glyph === T.QUESTION) {
    const old = lv.contents.get(key(x, y));
    if (!old || old.hidden) setContent(lv, p, x, y, { ...(content || defaultContent()), hidden: false });
  } else if (lv.contents.has(key(x, y)) && el.glyph !== T.BRICK) {
    // A block content only lives on a question or brick tile.
    setContent(lv, p, x, y, null);
  }
}

export const defaultContent = () => ({ kind: 'coin', count: 1, hidden: false, pool: null });

// ---------------------------------------------------------------- drawing

/** Every cell on the line between two cells, with no gaps: a fast drag of the
 * pen must not skip cells (spec level-editor / 拖动连续绘制). Consecutive cells
 * always share an edge — a diagonal-only step would leave a gap in a floor that
 * a player could fall through. */
export function line(x0, y0, x1, y1) {
  const out = [[x0, y0]];
  const dx = Math.abs(x1 - x0), dy = Math.abs(y1 - y0);
  const sx = Math.sign(x1 - x0), sy = Math.sign(y1 - y0);
  let x = x0, y = y0, ix = 0, iy = 0;
  while (ix < dx || iy < dy) {
    // Step along whichever axis is further behind the ideal line.
    if ((0.5 + ix) * dy < (0.5 + iy) * dx) { x += sx; ix++; } else { y += sy; iy++; }
    out.push([x, y]);
  }
  return out;
}

/** The cells of the rectangle with corners at two cells. */
export function rect(x0, y0, x1, y1) {
  const out = [];
  const [ax, bx] = [Math.min(x0, x1), Math.max(x0, x1)];
  const [ay, by] = [Math.min(y0, y1), Math.max(y0, y1)];
  for (let y = ay; y <= by; y++) for (let x = ax; x <= bx; x++) out.push([x, y]);
  return out;
}

// ---------------------------------------------------------------- ground

/** Fill the bottom of the level (or columns x0..x1) with ground, `layers`
 * deep. With keepGaps, a column whose bottom row is not ground stays a gap
 * (spec level-editor / 一键铺地面). The turf is drawn by the game on the top
 * layer of ground, so nothing else is needed.
 *
 * It only fills open air: a pipe, a staircase or a block already standing in
 * those rows is left whole. And nothing ends up buried: the spawn point and
 * any enemy the new ground would cover are lifted to stand on top of it. */
export function fillGround(lv, { layers = 2, keepGaps = false, x0 = 0, x1 = lv.width - 1 } = {}, p = newPatch('铺地面')) {
  layers = Math.max(1, Math.min(4, layers));
  for (let x = Math.max(0, x0); x <= Math.min(lv.width - 1, x1); x++) {
    if (keepGaps && at(lv, x, HEIGHT - 1) !== T.GROUND) continue;
    for (let y = HEIGHT - layers; y < HEIGHT; y++) {
      if (at(lv, x, y) !== T.AIR && at(lv, x, y) !== T.COIN) continue;
      if (lv.contents.has(key(x, y))) continue; // a hidden block lives in the air here
      setTile(lv, p, x, y, T.GROUND);
    }
  }
  liftOutOfGround(lv, p);
  return p;
}

/** Move the spawn and enemies that are inside solid ground up to the first
 * open cell above. */
function liftOutOfGround(lv, p) {
  const solid = (x, y) => { const ch = at(lv, x, y); return ch && ch !== T.AIR && ch !== T.COIN && !isFlagGlyph(ch); };
  const up = (x, y) => { while (y > 0 && solid(x, y)) y--; return y; };
  if (solid(lv.spawn.x, lv.spawn.y)) {
    const y = up(lv.spawn.x, lv.spawn.y);
    changeField(lv, p, 'spawn', s => { s.y = y; });
  }
  if (lv.entities.some(e => solid(e.x, e.y))) {
    snapEntities(lv, p);
    for (const e of lv.entities) if (solid(e.x, e.y)) e.y = up(e.x, e.y);
    finishEntities(lv, p);
  }
}

// ---------------------------------------------------------------- stamps

/** The tiles a stamp would place, relative to its anchor (the cell clicked).
 * Returns [{dx, dy, glyph}] — or for 'gap', glyph null meaning "clear". */
export function stampCells(kind, size, lv, x, y) {
  const out = [];
  switch (kind) {
    case 'pipe': {
      // The anchor is the left half; the pipe stands on whatever is below
      // the click, so it grows up from the ground.
      const h = Math.max(1, size);
      const base = groundBelow(lv, x, y);
      const top = base - h;
      for (let row = top; row < base; row++) {
        const lip = row === top;
        out.push({ dx: 0, dy: row - y, glyph: lip ? T.PIPE_LIP_L : T.PIPE_BODY_L });
        out.push({ dx: 1, dy: row - y, glyph: lip ? T.PIPE_LIP_R : T.PIPE_BODY_R });
      }
      break;
    }
    case 'stairs-up':
    case 'stairs-down': {
      const n = Math.max(1, size);
      const base = groundBelow(lv, x, y);
      for (let i = 0; i < n; i++) {
        const h = kind === 'stairs-up' ? i + 1 : n - i;
        for (let k = 0; k < h; k++) out.push({ dx: i, dy: base - 1 - k - y, glyph: T.STAIR });
      }
      break;
    }
    case 'gap': {
      for (let i = 0; i < Math.max(1, size); i++) {
        for (let row = 0; row < HEIGHT; row++) {
          if (at(lv, x + i, row) === T.GROUND) out.push({ dx: i, dy: row - y, glyph: null });
        }
      }
      break;
    }
    case 'brick-row':
      for (let i = 0; i < Math.max(1, size); i++) out.push({ dx: i, dy: 0, glyph: T.BRICK });
      break;
    case 'coin-arc': {
      // An arch of coins, the shape a jump traces.
      const n = Math.max(2, size);
      for (let i = 0; i < n; i++) {
        const t = n === 1 ? 0 : i / (n - 1);
        const lift = Math.round(Math.sin(t * Math.PI) * Math.min(3, Math.floor(n / 2)));
        out.push({ dx: i, dy: -lift, glyph: T.COIN });
      }
      break;
    }
  }
  return out;
}

/** The first solid row at or below (x, y) — where things stand. */
function groundBelow(lv, x, y) {
  for (let row = Math.max(0, y); row < HEIGHT; row++) {
    const ch = at(lv, x, row);
    if (ch && ch !== T.AIR && ch !== T.COIN && !isFlagGlyph(ch)) return row;
  }
  return HEIGHT;
}

export const STAMPS = [
  { id: 'pipe', label: '水管', size: 2, min: 1, max: 10, unit: '高' },
  { id: 'stairs-up', label: '上行阶梯', size: 4, min: 1, max: 10, unit: '级' },
  { id: 'stairs-down', label: '下行阶梯', size: 4, min: 1, max: 10, unit: '级' },
  { id: 'gap', label: '坑', size: 2, min: 1, max: 12, unit: '宽' },
  { id: 'brick-row', label: '一排砖', size: 4, min: 1, max: 16, unit: '块' },
  { id: 'coin-arc', label: '金币阵', size: 5, min: 2, max: 12, unit: '枚' },
];

/** Place a stamp. Pipes are always whole: both halves, lip on top. */
export function placeStamp(lv, p, kind, size, x, y) {
  for (const c of stampCells(kind, size, lv, x, y)) {
    const cx = x + c.dx, cy = y + c.dy;
    if (!inside(lv, cx, cy) || isFlagGlyph(at(lv, cx, cy))) continue;
    if (c.glyph === null) eraseCell(lv, p, cx, cy);
    else {
      setTile(lv, p, cx, cy, c.glyph);
      if (lv.contents.has(key(cx, cy)) && c.glyph !== T.BRICK) setContent(lv, p, cx, cy, null);
    }
  }
}

// ---------------------------------------------------------------- selection

/** Copy a rectangle: tiles, block contents and enemies, relative to its
 * top-left (spec level-editor / 选区与剪贴板). */
export function copyRect(lv, r) {
  const clip = { w: r.x1 - r.x0 + 1, h: r.y1 - r.y0 + 1, tiles: [], contents: [], entities: [] };
  for (let y = r.y0; y <= r.y1; y++) {
    for (let x = r.x0; x <= r.x1; x++) {
      const ch = at(lv, x, y);
      clip.tiles.push(isFlagGlyph(ch) ? T.AIR : ch);
      const c = lv.contents.get(key(x, y));
      if (c) clip.contents.push({ dx: x - r.x0, dy: y - r.y0, c: structuredClone(c) });
    }
  }
  for (const e of lv.entities) {
    if (e.x >= r.x0 && e.x <= r.x1 && e.y >= r.y0 && e.y <= r.y1) {
      clip.entities.push({ kind: e.kind, dx: e.x - r.x0, dy: e.y - r.y0 });
    }
  }
  return clip;
}

/** Clear a rectangle. */
export function clearRect(lv, p, r) {
  for (let y = r.y0; y <= r.y1; y++) for (let x = r.x0; x <= r.x1; x++) eraseCell(lv, p, x, y);
}

/** Paste a clip with its top-left at (x, y). Air in the clip overwrites. */
export function paste(lv, p, clip, x, y) {
  for (let dy = 0; dy < clip.h; dy++) {
    for (let dx = 0; dx < clip.w; dx++) {
      const cx = x + dx, cy = y + dy;
      if (!inside(lv, cx, cy) || isFlagGlyph(at(lv, cx, cy))) continue;
      eraseCell(lv, p, cx, cy);
      setTile(lv, p, cx, cy, clip.tiles[dy * clip.w + dx]);
    }
  }
  for (const { dx, dy, c } of clip.contents) {
    if (inside(lv, x + dx, y + dy)) setContent(lv, p, x + dx, y + dy, c);
  }
  if (clip.entities.length) {
    snapEntities(lv, p);
    for (const e of clip.entities) {
      if (inside(lv, x + e.dx, y + e.dy)) lv.entities.push({ kind: e.kind, x: x + e.dx, y: y + e.dy });
    }
    finishEntities(lv, p);
  }
}

/** Move a rectangle's contents by (dx, dy). */
export function moveRect(lv, p, r, dx, dy) {
  const clip = copyRect(lv, r);
  clearRect(lv, p, r);
  paste(lv, p, clip, r.x0 + dx, r.y0 + dy);
}

// ---------------------------------------------------------------- spawn & flag

/** Write the flagpole's tiles for lv.flag. */
export function placeFlagTiles(lv, p = null) {
  const set = p ? (x, y, ch) => setTile(lv, p, x, y, ch) : (x, y, ch) => { lv.tiles[y * lv.width + x] = ch; };
  const f = lv.flag;
  set(f.x, f.topY, T.BALL);
  for (let y = f.topY + 1; y <= f.bottomY; y++) set(f.x, y, T.POLE);
}

/** Move the one flagpole to a column; the old one goes (spec level-editor /
 * 出生点与旗杆唯一). */
export function moveFlag(lv, p, x) {
  if (x < 0 || x >= lv.width) return;
  const old = lv.flag;
  for (let y = 0; y < HEIGHT; y++) if (isFlagGlyph(at(lv, old.x, y))) setTile(lv, p, old.x, y, T.AIR);
  changeField(lv, p, 'flag', f => { f.x = x; });
  for (let y = lv.flag.topY; y <= lv.flag.bottomY; y++) eraseCell(lv, p, x, y);
  placeFlagTiles(lv, p);
}

/** Change the pole's height: its top row, keeping its foot. */
export function setFlagTop(lv, p, topY) {
  topY = Math.max(0, Math.min(lv.flag.bottomY - 1, topY));
  for (let y = 0; y < HEIGHT; y++) if (isFlagGlyph(at(lv, lv.flag.x, y))) setTile(lv, p, lv.flag.x, y, T.AIR);
  changeField(lv, p, 'flag', f => { f.topY = topY; });
  placeFlagTiles(lv, p);
}

// ---------------------------------------------------------------- width

/** Columns that a resize would delete and that hold something. */
export function lostByResize(lv, newWidth, side) {
  const lost = [];
  const n = lv.width - newWidth;
  if (n <= 0) return lost;
  const cols = side === 'left' ? range(0, n) : range(newWidth, lv.width);
  for (const x of cols) {
    let busy = false;
    for (let y = 0; y < HEIGHT; y++) {
      const ch = at(lv, x, y);
      if (ch !== T.AIR && !(y >= HEIGHT - 4 && ch === T.GROUND)) busy = true;
    }
    if (lv.entities.some(e => e.x === x) || lv.spawn.x === x || lv.flag.x === x) busy = true;
    if (busy) lost.push(x);
  }
  return lost;
}

function range(a, b) { const out = []; for (let i = a; i < b; i++) out.push(i); return out; }

/** Change the width, adding or removing columns on one side. Everything keeps
 * its place relative to the side that does not move. */
export function resize(lv, p, newWidth, side = 'right') {
  newWidth = Math.max(16, Math.min(1000, Math.round(newWidth)));
  if (newWidth === lv.width) return;
  // Resizing rebuilds every array; record the whole level, which is simplest
  // and still small (a few thousand cells).
  const before = snapshot(lv);
  const shift = side === 'left' ? newWidth - lv.width : 0;
  const tiles = new Array(newWidth * HEIGHT).fill(T.AIR);
  for (let y = 0; y < HEIGHT; y++) {
    for (let x = 0; x < lv.width; x++) {
      const nx = x + shift;
      if (nx >= 0 && nx < newWidth) tiles[y * newWidth + nx] = lv.tiles[y * lv.width + x];
    }
  }
  // New columns on the right get ground under them, continuing the floor.
  if (side === 'right') {
    for (let x = lv.width; x < newWidth; x++) {
      for (let y = 0; y < HEIGHT; y++) {
        if (at(lv, lv.width - 1, y) === T.GROUND) tiles[y * newWidth + x] = T.GROUND;
      }
    }
  }
  const contents = new Map();
  for (const [k, c] of lv.contents) {
    const [x, y] = k.split(',').map(Number);
    if (x + shift >= 0 && x + shift < newWidth) contents.set(key(x + shift, y), c);
  }
  lv.tiles = tiles;
  lv.contents = contents;
  lv.entities = lv.entities.filter(e => e.x + shift >= 0 && e.x + shift < newWidth).map(e => ({ ...e, x: e.x + shift }));
  lv.width = newWidth;
  lv.spawn.x = clamp(lv.spawn.x + shift, 0, newWidth - 1);
  const fx = clamp(lv.flag.x + shift, 0, newWidth - 1);
  if (fx !== lv.flag.x + shift) {
    lv.flag.x = fx;
    placeFlagTiles(lv);
  } else lv.flag.x = fx;
  p.width = [before, snapshot(lv)];
}

const clamp = (v, a, b) => Math.max(a, Math.min(b, v));

function snapshot(lv) {
  return {
    width: lv.width, tiles: lv.tiles.slice(), contents: structuredClone(lv.contents),
    entities: structuredClone(lv.entities), spawn: { ...lv.spawn }, flag: { ...lv.flag },
  };
}

function resizeRaw(lv, snap) {
  lv.width = snap.width; lv.tiles = snap.tiles.slice();
  lv.contents = structuredClone(snap.contents); lv.entities = structuredClone(snap.entities);
  lv.spawn = { ...snap.spawn }; lv.flag = { ...snap.flag };
}

// ---------------------------------------------------------------- random

/** Each pool entry's chance, in percent, rounded so the shown numbers add to
 * 100 (spec random-block-content / 随机块在编辑器中的配置). */
export function probabilities(pool) {
  const total = pool.reduce((s, p) => s + Math.max(0, p.w), 0);
  if (total <= 0) return pool.map(() => 0);
  const exact = pool.map(p => (Math.max(0, p.w) / total) * 100);
  const floors = exact.map(Math.floor);
  let left = 100 - floors.reduce((a, b) => a + b, 0);
  const order = exact.map((v, i) => [v - floors[i], i]).sort((a, b) => b[0] - a[0]);
  for (const [, i] of order) { if (left <= 0) break; if (pool[i].w > 0) { floors[i]++; left--; } }
  return floors;
}

/** Set the same content on many blocks at once, as one change. */
export function setContents(lv, p, cells, content) {
  for (const [x, y] of cells) {
    const old = lv.contents.get(key(x, y));
    if (!old) continue;
    setContent(lv, p, x, y, { ...structuredClone(content), hidden: old.hidden });
  }
}

/** The block cells (with contents) inside a rectangle. */
export function blocksIn(lv, r) {
  const out = [];
  for (const k of lv.contents.keys()) {
    const [x, y] = k.split(',').map(Number);
    if (x >= r.x0 && x <= r.x1 && y >= r.y0 && y <= r.y1) out.push([x, y]);
  }
  return out;
}

// ---------------------------------------------------------------- ids

/** Turn a title into a community level id, c-author-name. */
export function suggestId(author, title) {
  const slug = s => (s || '').toLowerCase().normalize('NFKD').replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
  const a = slug(author) || 'me';
  const t = slug(title) || 'level';
  return `c-${a}-${t}`.slice(0, 64);
}

export const validId = id => /^[a-z0-9][a-z0-9-]{0,63}$/.test(id) &&
  (/^[0-9]+-[0-9]+$/.test(id) || /^c-[a-z0-9]+(-[a-z0-9]+)+$/.test(id));

/** Why an id is not valid, as an i18n key — the same two reasons the server
 * gives. */
validId.reason = id => (/^[a-z0-9][a-z0-9-]{0,63}$/.test(id) ? 'err.bad-id-form' : 'err.bad-id-chars');
