// app.js — the level editor's page: canvas, tools, panels, saving, playtest.
//
// All level logic lives in model.js (tested in Node); everything that must
// match the game — sprites, the sky, checking, formatting, the playtest — comes
// from the game's own Go code through WebAssembly (globalThis.mario).

import * as M from './model.js';
import { t, issueText, applyStatic, setLang, current as currentLang, LANGS } from './i18n.js';
import { readDraft, storeDraft, deleteDraft } from './drafts.js';

// ================================================================ state

const ART = 4;            // art pixels per tile: what the terminal shows by default

const S = {
  lv: M.starter(),
  id: '',              // the level's file id
  // source is the folder the level is kept in: 'builtin' or 'local'. A new
  // level gets the default for where the editor runs.
  source: '',
  openedAs: '',        // the key it was opened from
  baseRevision: '',    // the version of the file this document was opened from
  revision: 0,        // increments whenever the editor document changes
  draftToken: '',
  saving: false,
  opening: false,
  history: new M.History(),
  dirty: false,
  info: null,          // /api/info: folders and what can be written

  tool: 'hand',        // hand | pen | rect | erase | pick | select | stamp
  element: 'ground',
  stamp: 'pipe',
  stampSize: { ...Object.fromEntries(M.STAMPS.map(s => [s.id, s.size])) },
  content: M.defaultContent(),   // what a new question block holds

  // view: the tile at the canvas's left edge, and pixels per tile
  view: { x: 0, y: 0, scale: 40 },
  hover: null,         // [x, y] under the mouse
  selection: null,     // {x0,y0,x1,y1}
  selected: null,      // {type:'block'|'spawn'|'flag', x, y}
  clipboard: null,
  pastePreview: false,
  drag: null,          // the pointer gesture in progress
  grid: true,

  sprites: new Map(),  // id -> {canvas, ox, oy, w, h}
  sky: [],             // one colour per art row
  issues: [],
  trace: null,         // {points, death, stale, hidden}
  palettes: null,

  playing: null,       // the playtest in progress
  playOpts: { startBig: false, invincible: false, fixedSeed: false, seed: 1, showDraws: false },
};

const $ = id => document.getElementById(id);
const canvas = $('canvas');
const ctx = canvas.getContext('2d');
const mini = $('minimap');
const mctx = mini.getContext('2d');

// ================================================================ boot

async function boot() {
  setLang(currentLang());
  applyStatic();
  try {
    await loadWasm();
  } catch (e) {
    $('loading').classList.add('failed');
    $('loading-msg').textContent = t('loading.failed', { e: e.message });
    return;
  }
  S.palettes = JSON.parse(mario.palettes());
  S.info = await (await fetch('api/info')).json();
  $('help-version').textContent = S.info.version || '';

  buildLang(); buildPanels();
  wire();

  const want = location.hash.slice(1) || S.info.open;
  if (!(await restoreDraft(want || '')) && !(want && await openLevel(want))) {
    loadDoc(M.starter(), '', S.info.newSource, '', false);
  }

  $('loading').remove();
  resize();
}

/** Build every panel whose text depends on the language. Called again when
 * the language changes; the level and the undo history are untouched. */
function buildPanels() {
  for (const id of ['tools', 'elements', 'themes', 'new-themes', 'look-rows', 'help-body']) $(id).innerHTML = '';
  buildTools(); buildElements(); buildStamps(); buildLook(); buildHelp();
  paintPaletteIcons();
}

function buildLang() {
  const sel = $('lang');
  for (const l of LANGS) {
    const o = document.createElement('option'); o.value = l.id; o.textContent = l.label;
    sel.append(o);
  }
  sel.value = currentLang();
  sel.onchange = () => {
    setLang(sel.value);
    applyStatic();
    // Messages already on screen were written in the old language.
    $('toast').hidden = true; $('toast').textContent = '';
    $('st-draft').textContent = '';
    if ($('dlg-open').open) $('dlg-open').close();
    $('open-list').innerHTML = '';
    $('new-where').textContent = '';
    buildPanels();
    syncTools(); syncLook(); syncProps(); syncLevelPanel(); renderIssues(); draw();
  };
}

async function loadWasm() {
  const go = new Go();
  const ready = new Promise(r => { globalThis.marioReady = r; });
  const resp = await fetch('editor.wasm');
  if (!resp.ok) throw new Error('editor.wasm missing');
  const total = +resp.headers.get('Content-Length') || 4e6;
  const reader = resp.body.getReader();
  const chunks = []; let got = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value); got += value.length;
    $('loading-fill').style.width = Math.min(100, got / total * 100) + '%';
  }
  const bytes = new Uint8Array(got); let o = 0;
  for (const c of chunks) { bytes.set(c, o); o += c.length; }
  const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
  go.run(instance);
  await ready;
}

// ================================================================ document

function loadDoc(lv, id, source, openedAs, dirty, baseRevision = '', draftToken = crypto.randomUUID()) {
  S.lv = lv; S.id = id; S.source = source; S.openedAs = openedAs;
  S.baseRevision = baseRevision; S.draftToken = draftToken; S.revision++;
  S.history = new M.History();
  S.selection = null; S.selected = null; S.trace = null;
  setDirty(dirty);
  refreshStyle();
  syncLevelPanel();
  syncProps();
  S.view.x = Math.max(0, lv.spawn.x - 4); S.view.y = 0;
  if (resize.done) S.view.scale = defaultScale();
  fitHeight();
  check();
  draw();
}

/** Split a level key ("local:c-a-b" or "1-1") into its folder and id. */
function parseKey(key) {
  return key.startsWith('local:') ? ['local', key.slice(6)] : ['builtin', key];
}
const keyOf = (source, id) => (source === 'local' ? 'local:' : '') + id;

async function openLevel(key) {
  const [source, id] = parseKey(key);
  try {
    const r = await fetch('api/levels/' + source + '/' + encodeURIComponent(id));
    if (!r.ok) {
      toast(t('toast.notFound', { id }), 4000, true);
      return false;
    }
    const lv = M.fromFile(await r.text());
    clearDraft();
    loadDoc(lv, id, source, key, false, r.headers.get('ETag') || '');
    history.replaceState(null, '', '#' + key);
    return true;
  } catch (error) {
    toast(t(error instanceof RangeError ? 'toast.unsupportedHeight' : 'toast.openFailed', { id, height: M.HEIGHT }), 5000, true);
    return false;
  }
}

/** Run an edit as one undoable step. fn receives the patch. */
function edit(label, fn) {
  const p = M.newPatch(label);
  fn(p);
  commit(p);
}

function commit(p) {
  if (!S.history.push(p)) return;
  afterChange(p);
}

function afterChange(p) {
  setDirty(true);
  if (p && (p.style || p.width)) refreshStyle();
  if (S.trace) S.trace.stale = true;
  syncLevelPanel();
  syncProps();
  check();
  draw();
}

function undo() { const p = S.history.undo(S.lv); if (p) { afterChange(p); toast(t('toast.undo', { x: p.label }), 1200); } }
function redo() { const p = S.history.redo(S.lv); if (p) { afterChange(p); toast(t('toast.redo', { x: p.label }), 1200); } }

function setDirty(d) {
  S.dirty = d;
  $('dirty').hidden = !d;
  $('btn-undo').disabled = !S.history.canUndo;
  $('btn-redo').disabled = !S.history.canRedo;
  if (d) {
    S.revision++;
    saveDraftSoon();
  }
}

// ================================================================ checking

let checkTimer = 0;
function check() {
  clearTimeout(checkTimer);
  checkTimer = setTimeout(() => {
    S.issues = JSON.parse(mario.validate(M.toFile(S.lv)));
    renderIssues();
    draw();
  }, 120);
}

const errors = () => S.issues.filter(i => i.severity === 'error');

function renderIssues() {
  const ul = $('issues'); ul.innerHTML = '';
  const nErr = errors().length, nWarn = S.issues.length - nErr;
  const count = $('issue-count');
  count.textContent = S.issues.length ? t('check.count', { e: nErr, w: nWarn }) : '';
  count.className = 'count' + (nErr ? ' bad' : '');
  if (!S.issues.length) {
    const li = document.createElement('li'); li.className = 'ok'; li.textContent = t('check.none');
    ul.append(li); return;
  }
  for (const is of S.issues) {
    const li = document.createElement('li');
    li.className = is.severity;
    const where = is.x >= 0 ? (is.y >= 0 ? t('check.atRow', { x: is.x, y: is.y }) : t('check.at', { x: is.x })) : t('check.whole');
    li.innerHTML = `<span class="dot"></span><span><span class="msg"></span><br><span class="at"></span></span>`;
    li.querySelector('.msg').textContent = issueText(is);
    li.querySelector('.at').textContent = where;
    li.onclick = () => focusCell(is.x, is.y);
    ul.append(li);
  }
}

/** Scroll the canvas to a cell and flash it (spec level-contribution / 点击定位). */
function focusCell(x, y) {
  if (x < 0) return;
  const cols = canvas.width / S.view.scale;
  S.view.x = clampView(x - cols / 2);
  S.flash = { x, y: y < 0 ? null : y, until: performance.now() + 1500 };
  animateFlash();
}

function animateFlash() {
  draw();
  if (S.flash && performance.now() < S.flash.until) requestAnimationFrame(animateFlash);
  else { S.flash = null; draw(); }
}

// ================================================================ style & sprites

/** Rebuild the sprite atlas and sky for the level's current look. */
function refreshStyle() {
  const file = M.toFile(S.lv);
  S.sprites.clear();
  for (const sp of mario.sprites(file, ART)) {
    const c = document.createElement('canvas');
    c.width = sp.w; c.height = sp.h;
    c.getContext('2d').putImageData(new ImageData(sp.rgba, sp.w, sp.h), 0, 0);
    S.sprites.set(sp.id, { canvas: c, ox: sp.ox, oy: sp.oy, w: sp.w, h: sp.h });
  }
  S.sky = JSON.parse(mario.sky(file, ART));
  // Stars, for a level at night: the game's own field, drawn over the sky.
  S.starFile = file;
  S.stars = JSON.parse(mario.stars(file, starStep()) || '[]') || [];
  twinkleStars();
  miniColours.cache = null;
  paintPaletteIcons();
  syncLook();
}

/** The game's simulation step for "now", at 60 steps a second, so the canvas
 * twinkles at the game's pace. */
function starStep() { return Math.floor(performance.now() * 60 / 1000); }

/** Keep a night sky twinkling: ask the game for the stars' colours at the
 * current step about thirty times a second, and repaint when any changed. The
 * loop stops by itself when the level is no longer at night, and does nothing
 * while a playtest is on (the playtest draws its own frames). */
function twinkleStars() {
  if (twinkleStars.timer || !S.stars || !S.stars.length) return;
  twinkleStars.timer = setInterval(() => {
    if (!S.stars || !S.stars.length) {
      clearInterval(twinkleStars.timer); twinkleStars.timer = 0; return;
    }
    if (S.playing || document.hidden) return;
    const next = JSON.parse(mario.stars(S.starFile, starStep()) || '[]') || [];
    const changed = next.length !== S.stars.length || next.some((s, i) => s.Color !== S.stars[i].Color);
    S.stars = next;
    if (changed) draw();
  }, 33);
}

/** Draw a sprite into a small canvas (palette icons, property panel). */
function iconCanvas(id, size = 24) {
  const c = document.createElement('canvas');
  c.width = size; c.height = size;
  c.dataset.sprite = id;
  paintIcon(c);
  return c;
}

function paintIcon(c) {
  const g = c.getContext('2d');
  g.clearRect(0, 0, c.width, c.height);
  const sp = S.sprites.get(c.dataset.sprite);
  if (!sp) return;
  g.imageSmoothingEnabled = false;
  const k = Math.floor(c.width / Math.max(sp.w, sp.h, ART)) || 1;
  g.drawImage(sp.canvas, (c.width - sp.w * k) / 2, (c.height - sp.h * k) / 2, sp.w * k, sp.h * k);
}

function paintPaletteIcons() {
  document.querySelectorAll('canvas[data-sprite]').forEach(paintIcon);
}

// ================================================================ view

function resize() {
  const wrap = $('canvas-wrap');
  const first = !resize.done;
  resize.done = true;
  canvas.width = wrap.clientWidth; canvas.height = wrap.clientHeight;
  const pc = $('play-canvas'); pc.width = wrap.clientWidth; pc.height = wrap.clientHeight;
  mini.width = mini.clientWidth; mini.height = mini.clientHeight;
  if (first) S.view.scale = defaultScale();
  fitHeight();
  draw();
}

/** The closest the view zooms out: small enough to see most of a level at
 * once, never so small a tile is a speck. */
function minScale() { return 8; }

/** A comfortable starting scale: the whole height of the level with a little
 * room above it for the ruler, so as much of the level as possible shows. */
function defaultScale() { return Math.max(16, Math.min(56, Math.floor((canvas.height - 24) / M.HEIGHT))); }

function fitHeight() {
  if (!canvas.height) return;
  if (S.view.scale < minScale()) S.view.scale = minScale();
  S.view.x = clampView(S.view.x);
  S.view.y = clampViewY(S.view.y);
}

function clampView(x) {
  const cols = canvas.width / S.view.scale;
  return Math.max(-2, Math.min(S.lv.width - cols + 2, x));
}
/** Vertically the level is short; when it fits, keep it pinned to the bottom
 * of the canvas (where the ground is), otherwise let it scroll. */
function clampViewY(y) {
  const rows = canvas.height / S.view.scale;
  if (rows >= M.HEIGHT) return M.HEIGHT - rows;
  return Math.max(0, Math.min(M.HEIGHT - rows, y));
}

function cellAt(ev) {
  const r = canvas.getBoundingClientRect();
  const px = (ev.clientX - r.left) * (canvas.width / r.width);
  const py = (ev.clientY - r.top) * (canvas.height / r.height);
  return [Math.floor(S.view.x + px / S.view.scale), Math.floor(S.view.y + py / S.view.scale)];
}

const tileToPx = (x, y) => [(x - S.view.x) * S.view.scale, (y - S.view.y) * S.view.scale];

/** Zoom keeping a canvas point fixed (spec level-editor / 视图导航). */
function zoomAt(factor, px, py) {
  const s0 = S.view.scale;
  const s1 = Math.max(minScale(), Math.min(96, Math.round(s0 * factor)));
  if (s1 === s0) return;
  const tx = S.view.x + px / s0, ty = S.view.y + py / s0;
  S.view.scale = s1;
  S.view.x = clampView(tx - px / s1);
  S.view.y = clampViewY(ty - py / s1);
  draw();
}

// ================================================================ drawing

let drawQueued = false;
function draw() {
  if (drawQueued) return;
  drawQueued = true;
  requestAnimationFrame(() => { drawQueued = false; paint(); paintMini(); status(); });
}

function paint() {
  const lv = S.lv, sc = S.view.scale;
  const W = canvas.width, H = canvas.height;
  ctx.imageSmoothingEnabled = false;

  // Outside the level: a dark, plain ground so the level's edges are clear.
  ctx.fillStyle = '#16181d';
  ctx.fillRect(0, 0, W, H);

  // Sky: the game's own colours, one band per art row.
  const rowH = sc / ART;
  const [lx] = tileToPx(0, 0), [rx] = tileToPx(lv.width, 0);
  for (let r = 0; r < S.sky.length; r++) {
    const y = (r / ART - S.view.y) * sc;
    if (y > H || y + rowH < 0) continue;
    ctx.fillStyle = S.sky[r];
    ctx.fillRect(Math.max(0, lx), Math.floor(y), Math.min(W, rx) - Math.max(0, lx), Math.ceil(rowH) + 1);
  }

  // Stars on a night sky, where the game puts them and twinkling by the same
  // rule (see twinkleStars). One art pixel each, so they scale with the zoom.
  if (S.stars && S.stars.length) {
    const px = Math.max(1, Math.round(rowH));
    for (const s of S.stars) {
      if (!s.Color) continue; // out for a moment
      const sx = (s.X / ART - S.view.x) * sc, sy = (s.Y / ART - S.view.y) * sc;
      if (sx < -px || sx > W || sy < -px || sy > H) continue;
      ctx.fillStyle = s.Color;
      ctx.fillRect(Math.round(sx), Math.round(sy), px, px);
    }
  }

  const x0 = Math.max(0, Math.floor(S.view.x)), x1 = Math.min(lv.width - 1, Math.ceil(S.view.x + W / sc));
  const y0 = Math.max(0, Math.floor(S.view.y)), y1 = Math.min(M.HEIGHT - 1, Math.ceil(S.view.y + H / sc));
  const k = sc / ART;

  const blit = (id, x, y) => {
    const sp = S.sprites.get(id); if (!sp) return;
    const [px, py] = tileToPx(x, y);
    ctx.drawImage(sp.canvas, Math.round(px + sp.ox * k), Math.round(py + sp.oy * k), Math.round(sp.w * k), Math.round(sp.h * k));
  };

  for (let y = y0; y <= y1; y++) {
    for (let x = x0; x <= x1; x++) {
      const ch = M.at(lv, x, y);
      const id = spriteFor(ch, x, y);
      if (id) blit(id, x, y);
    }
  }

  // Flagpole: drawn as the game draws it would need the whole world; a
  // plain pole and ball read clearly at every zoom.
  drawFlag();

  // Hidden blocks: a faint outline, they have no tile.
  for (const [kk, c] of lv.contents) {
    const [x, y] = kk.split(',').map(Number);
    if (c.hidden) {
      const [px, py] = tileToPx(x, y);
      ctx.save(); ctx.globalAlpha = .45; blit('question', x, y); ctx.restore();
      ctx.strokeStyle = '#fff'; ctx.setLineDash([3, 3]); ctx.strokeRect(px + 1.5, py + 1.5, sc - 3, sc - 3); ctx.setLineDash([]);
    }
    markContent(x, y, c);
  }

  for (const e of lv.entities) blit(e.kind, e.x, e.y);
  drawSpawn();

  if (S.grid && sc >= 12) drawGrid(x0, x1, y0, y1);
  drawRulers();
  if (S.trace && !S.trace.hidden) drawTrace();
  drawSelection();
  drawPreview();
  drawIssueMarks();
  if (S.flash) {
    const fade = (S.flash.until - performance.now()) / 1500;
    const [px, py] = tileToPx(S.flash.x, S.flash.y ?? 0);
    ctx.strokeStyle = `rgba(255,220,0,${fade})`; ctx.lineWidth = 3;
    if (S.flash.y == null) ctx.strokeRect(px, 0, sc, H); else ctx.strokeRect(px - 2, py - 2, sc + 4, sc + 4);
    ctx.lineWidth = 1;
  }
}

/** Which atlas sprite draws a tile. Ground with ground above it is plain soil,
 * exactly the game's rule, so the turf appears only on top. */
function spriteFor(ch, x, y) {
  switch (ch) {
    case 'X': return M.at(S.lv, x, y - 1) === 'X' ? 'ground-below' : 'ground';
    case 'H': return 'stair';
    case 'B': return 'brick';
    case '?': return 'question';
    case '[': return 'pipe-lip-l';
    case ']': return 'pipe-lip-r';
    case '{': return 'pipe-body-l';
    case '}': return 'pipe-body-r';
    case 'o': return 'coin';
  }
  return null;
}

/** A small badge on a block saying what it holds: the item's own sprite, a
 * coin count, or a die for a random block (spec level-editor / 属性面板). */
function markContent(x, y, c) {
  const sc = S.view.scale;
  if (sc < 16) return;
  const [px, py] = tileToPx(x, y);
  const b = Math.max(10, Math.round(sc * 0.45));
  const bx = px + sc - b + 2, by = py - 2;
  ctx.fillStyle = 'rgba(0,0,0,.55)';
  ctx.fillRect(bx - 1, by - 1, b + 2, b + 2);
  if (c.kind === 'random') {
    // A die: five pips on a white face.
    ctx.fillStyle = '#fff'; ctx.fillRect(bx, by, b, b);
    ctx.fillStyle = '#222';
    const d = Math.max(2, Math.round(b / 5));
    for (const [u, v] of [[1, 1], [3, 3], [1, 3], [3, 1], [2, 2]]) ctx.fillRect(bx + u * b / 5 - d / 2 + b / 10, by + v * b / 5 - d / 2 + b / 10, d, d);
    return;
  }
  if (c.kind === 'coin') {
    const sp = S.sprites.get('coin');
    if (sp) {
      const h = b, w = sp.w * h / sp.h;
      ctx.drawImage(sp.canvas, bx + (b - w) / 2, by, w, h);
    }
    if (c.count > 1) {
      ctx.fillStyle = '#fff'; ctx.font = `bold ${Math.round(b * .7)}px ui-monospace, Consolas, monospace`;
      ctx.textAlign = 'right'; ctx.textBaseline = 'top';
      ctx.fillText('×' + c.count, bx - 2, by);
    }
    return;
  }
  const sp = S.sprites.get('item-' + c.kind);
  if (sp) ctx.drawImage(sp.canvas, bx, by, b, b * sp.h / sp.w);
}

function drawFlag() {
  const f = S.lv.flag, sc = S.view.scale;
  const [px, py] = tileToPx(f.x, f.topY);
  const cx = px + sc / 2;
  ctx.fillStyle = '#b7bcc9';
  ctx.fillRect(cx - Math.max(1, sc / 16), py + sc / 2, Math.max(2, sc / 8), (f.bottomY - f.topY + 0.5) * sc);
  ctx.fillStyle = '#f5c842';
  ctx.beginPath(); ctx.arc(cx, py + sc / 2, Math.max(3, sc / 4), 0, Math.PI * 2); ctx.fill();
  ctx.fillStyle = '#f2f4f8';
  ctx.beginPath(); ctx.moveTo(cx, py + sc * .8); ctx.lineTo(cx - sc * .9, py + sc * 1.2); ctx.lineTo(cx, py + sc * 1.6); ctx.fill();
}

function drawSpawn() {
  const s = S.lv.spawn, sc = S.view.scale;
  const [px, py] = tileToPx(s.x, s.y);
  ctx.fillStyle = 'rgba(217,74,61,.85)';
  ctx.beginPath();
  ctx.moveTo(px + sc / 2, py + sc * .15);
  ctx.lineTo(px + sc * .85, py + sc * .55);
  ctx.lineTo(px + sc * .62, py + sc * .55);
  ctx.lineTo(px + sc * .62, py + sc * .95);
  ctx.lineTo(px + sc * .38, py + sc * .95);
  ctx.lineTo(px + sc * .38, py + sc * .55);
  ctx.lineTo(px + sc * .15, py + sc * .55);
  ctx.closePath(); ctx.fill();
  if (sc >= 20) {
    ctx.fillStyle = '#fff'; ctx.font = `bold ${Math.round(sc * .28)}px sans-serif`;
    ctx.textAlign = 'center'; ctx.textBaseline = 'bottom';
    ctx.fillText(t('spawn.badge'), px + sc / 2, py);
  }
}

function drawGrid(x0, x1, y0, y1) {
  const [, top] = tileToPx(0, 0), [, bottom] = tileToPx(0, M.HEIGHT);
  const [left] = tileToPx(0, 0), [right] = tileToPx(S.lv.width, 0);
  const gt = Math.max(0, top), gb = Math.min(canvas.height, bottom);
  const l = Math.max(0, left), r = Math.min(canvas.width, right);
  ctx.strokeStyle = 'rgba(255,255,255,.10)';
  ctx.beginPath();
  for (let x = x0; x <= x1 + 1; x++) { const [px] = tileToPx(x, 0); ctx.moveTo(Math.round(px) + .5, gt); ctx.lineTo(Math.round(px) + .5, gb); }
  for (let y = y0; y <= y1 + 1; y++) { const [, py] = tileToPx(0, y); ctx.moveTo(l, Math.round(py) + .5); ctx.lineTo(r, Math.round(py) + .5); }
  ctx.stroke();
  // Every tenth column a little stronger, to count by.
  ctx.strokeStyle = 'rgba(255,255,255,.22)';
  ctx.beginPath();
  for (let x = Math.ceil(x0 / 10) * 10; x <= x1 + 1; x += 10) { const [px] = tileToPx(x, 0); ctx.moveTo(Math.round(px) + .5, gt); ctx.lineTo(Math.round(px) + .5, gb); }
  ctx.stroke();
}

function drawRulers() {
  const sc = S.view.scale, W = canvas.width;
  // The column ruler sits just above the level's top row.
  const [, top] = tileToPx(0, 0);
  const ry = Math.max(0, Math.round(top) - 16);
  ctx.fillStyle = 'rgba(0,0,0,.55)'; ctx.fillRect(0, ry, W, 16);
  ctx.fillStyle = '#fff'; ctx.font = '11px ui-monospace, Consolas, monospace';
  ctx.textAlign = 'left'; ctx.textBaseline = 'top';
  const step = sc >= 30 ? 5 : sc >= 14 ? 10 : 20;
  for (let x = Math.ceil(S.view.x / step) * step; x < S.view.x + W / sc; x += step) {
    if (x < 0 || x >= S.lv.width) continue;
    const [px] = tileToPx(x, 0);
    ctx.fillText(String(x), px + 2, ry + 2);
  }
  if (S.hover) {
    const [px] = tileToPx(S.hover[0], 0);
    ctx.fillStyle = 'rgba(255,255,255,.25)'; ctx.fillRect(px, ry, sc, 16);
  }
}

function drawTrace() {
  const tr = S.trace, sc = S.view.scale;
  ctx.save();
  ctx.globalAlpha = tr.stale ? .35 : .9;
  ctx.strokeStyle = '#ffde59'; ctx.lineWidth = Math.max(2, sc / 12);
  ctx.setLineDash(tr.stale ? [6, 6] : []);
  ctx.beginPath();
  tr.points.forEach((p, i) => {
    const [px, py] = tileToPx(p.x, p.y);
    if (i === 0) ctx.moveTo(px, py); else ctx.lineTo(px, py);
  });
  ctx.stroke();
  ctx.setLineDash([]);
  ctx.fillStyle = '#4fc3f7';
  for (const p of tr.points) {
    if (!p.j) continue;
    const [px, py] = tileToPx(p.x, p.y);
    ctx.beginPath(); ctx.arc(px, py, Math.max(3, sc / 8), 0, Math.PI * 2); ctx.fill();
  }
  if (tr.death) {
    const [px, py] = tileToPx(tr.death.x, Math.min(tr.death.y, M.HEIGHT - .5));
    const r = Math.max(6, sc / 4);
    ctx.strokeStyle = '#ff5252'; ctx.lineWidth = 3;
    ctx.beginPath(); ctx.moveTo(px - r, py - r); ctx.lineTo(px + r, py + r); ctx.moveTo(px + r, py - r); ctx.lineTo(px - r, py + r); ctx.stroke();
  }
  if (tr.stale && sc >= 14) {
    ctx.globalAlpha = 1; ctx.fillStyle = 'rgba(0,0,0,.6)';
    const [px, py] = tileToPx(tr.points[0].x, tr.points[0].y);
    ctx.fillRect(px, py - 34, 64, 16);
    ctx.fillStyle = '#fff'; ctx.font = '11px sans-serif'; ctx.textBaseline = 'top';
    ctx.fillText(t('trace.stale'), px + 4, py - 32);
  }
  ctx.restore();
}

function drawSelection() {
  const sc = S.view.scale;
  const r = S.drag?.kind === 'select' ? normRect(S.drag.start, S.drag.cur) : S.selection;
  if (r) {
    const [px, py] = tileToPx(r.x0, r.y0);
    ctx.fillStyle = 'rgba(47,111,208,.15)';
    ctx.fillRect(px, py, (r.x1 - r.x0 + 1) * sc, (r.y1 - r.y0 + 1) * sc);
    ctx.strokeStyle = '#6a9ef0'; ctx.setLineDash([5, 4]); ctx.lineWidth = 2;
    ctx.strokeRect(px + 1, py + 1, (r.x1 - r.x0 + 1) * sc - 2, (r.y1 - r.y0 + 1) * sc - 2);
    ctx.setLineDash([]); ctx.lineWidth = 1;
  }
  if (S.selected) {
    const { x, y } = S.selected;
    const [px, py] = tileToPx(x, y);
    ctx.strokeStyle = '#fff'; ctx.lineWidth = 2; ctx.strokeRect(px - 1, py - 1, sc + 2, sc + 2);
    ctx.strokeStyle = '#2f6fd0'; ctx.strokeRect(px - 3, py - 3, sc + 6, sc + 6);
    ctx.lineWidth = 1;
  }
}

/** What would happen if the user clicked now: a translucent preview of the
 * stamp, the rectangle being dragged, the paste, or the cell under the pen. */
function drawPreview() {
  if (!S.hover || S.playing) return;
  const [hx, hy] = S.hover, sc = S.view.scale;
  ctx.save();
  ctx.globalAlpha = .55;
  if (S.pastePreview && S.clipboard) {
    const c = S.clipboard;
    for (let dy = 0; dy < c.h; dy++) for (let dx = 0; dx < c.w; dx++) {
      const id = spriteFor(c.tiles[dy * c.w + dx], -1, -1);
      if (id) blitAt(id, hx + dx, hy + dy);
    }
    for (const e of c.entities) blitAt(e.kind, hx + e.dx, hy + e.dy);
    ctx.globalAlpha = 1;
    const [px, py] = tileToPx(hx, hy);
    ctx.strokeStyle = '#6a9ef0'; ctx.setLineDash([4, 4]); ctx.strokeRect(px, py, c.w * sc, c.h * sc); ctx.setLineDash([]);
  } else if (S.tool === 'stamp') {
    for (const c of M.stampCells(S.stamp, S.stampSize[S.stamp], S.lv, hx, hy)) {
      const [px, py] = tileToPx(hx + c.dx, hy + c.dy);
      if (c.glyph === null) { ctx.fillStyle = 'rgba(255,60,60,.6)'; ctx.fillRect(px, py, sc, sc); }
      else blitAt(spriteFor(c.glyph, -1, -1), hx + c.dx, hy + c.dy);
    }
  } else if (S.drag?.kind === 'rect') {
    const r = normRect(S.drag.start, S.drag.cur);
    const el = M.elementById(S.element);
    for (const [x, y] of M.rect(r.x0, r.y0, r.x1, r.y1)) {
      if (S.drag.erase) { const [px, py] = tileToPx(x, y); ctx.fillStyle = 'rgba(255,60,60,.5)'; ctx.fillRect(px, py, sc, sc); }
      else if (el.sprite) blitAt(el.sprite, x, y);
    }
  } else if (S.tool === 'pen' && !S.drag) {
    const el = M.elementById(S.element);
    if (el.sprite) blitAt(el.sprite, hx, hy);
  }
  ctx.restore();
  // The cell under the mouse.
  const [px, py] = tileToPx(hx, hy);
  ctx.strokeStyle = S.tool === 'erase' ? '#ff6b6b' : 'rgba(255,255,255,.9)';
  ctx.lineWidth = 2; ctx.strokeRect(px + 1, py + 1, sc - 2, sc - 2); ctx.lineWidth = 1;
}

function blitAt(id, x, y) {
  const sp = S.sprites.get(id); if (!sp) return;
  const k = S.view.scale / ART;
  const [px, py] = tileToPx(x, y);
  ctx.drawImage(sp.canvas, Math.round(px + sp.ox * k), Math.round(py + sp.oy * k), Math.round(sp.w * k), Math.round(sp.h * k));
}

function drawIssueMarks() {
  const sc = S.view.scale;
  for (const is of S.issues) {
    if (is.x < 0 || is.y < 0) continue;
    const [px, py] = tileToPx(is.x, is.y);
    ctx.strokeStyle = is.severity === 'error' ? '#ff5252' : '#ffb74d';
    ctx.setLineDash([3, 2]); ctx.lineWidth = 2;
    ctx.strokeRect(px + 2, py + 2, sc - 4, sc - 4);
    ctx.setLineDash([]); ctx.lineWidth = 1;
  }
}

/** The whole level, small, with the visible window marked
 * (spec level-editor / 视图导航 小地图). */
function paintMini() {
  const lv = S.lv, W = mini.width, H = mini.height;
  if (!W || !H) return;
  const k = Math.min(W / lv.width, H / M.HEIGHT);
  const ox = (W - lv.width * k) / 2;
  mctx.fillStyle = getComputedStyle(document.body).getPropertyValue('--panel-2') || '#eee';
  mctx.fillRect(0, 0, W, H);
  mctx.fillStyle = S.sky[Math.floor(S.sky.length / 2)] || '#000';
  mctx.fillRect(ox, 0, lv.width * k, M.HEIGHT * k);
  // Each tile in its own sprite's main colour, so the minimap follows the
  // level's look.
  const colour = miniColours();
  for (let y = 0; y < M.HEIGHT; y++) for (let x = 0; x < lv.width; x++) {
    const ch = M.at(lv, x, y);
    const c = colour[ch === 'X' && M.at(lv, x, y - 1) !== 'X' ? 'turf' : ch];
    if (c) { mctx.fillStyle = c; mctx.fillRect(ox + x * k, y * k, Math.ceil(k), Math.ceil(k)); }
  }
  mctx.fillStyle = '#d94a3d';
  for (const e of lv.entities) mctx.fillRect(ox + e.x * k, e.y * k, Math.ceil(k), Math.ceil(k));
  for (const is of S.issues) {
    if (is.x < 0) continue;
    mctx.fillStyle = is.severity === 'error' ? '#ff5252' : '#ffb74d';
    mctx.fillRect(ox + is.x * k - 1, 0, 3, 3);
  }
  const vw = canvas.width / S.view.scale, vh = canvas.height / S.view.scale;
  mctx.strokeStyle = '#2f6fd0'; mctx.lineWidth = 2;
  mctx.strokeRect(ox + S.view.x * k, S.view.y * k, vw * k, Math.min(vh, M.HEIGHT) * k);
  mini._map = { k, ox };
}

/** The most common colour in each sprite, for the minimap. Cached until the
 * look changes. */
function miniColours() {
  if (miniColours.cache) return miniColours.cache;
  const main = id => {
    const sp = S.sprites.get(id); if (!sp) return null;
    const d = sp.canvas.getContext('2d').getImageData(0, 0, sp.w, sp.h).data;
    const n = new Map();
    for (let i = 0; i < d.length; i += 4) {
      if (d[i + 3] < 128) continue;
      const k = (d[i] << 16) | (d[i + 1] << 8) | d[i + 2];
      n.set(k, (n.get(k) || 0) + 1);
    }
    let best = 0, bestN = 0;
    for (const [k, c] of n) if (c > bestN) { best = k; bestN = c; }
    return '#' + best.toString(16).padStart(6, '0');
  };
  const pipe = main('pipe-body-l');
  miniColours.cache = {
    X: main('ground-below'), turf: main('ground'), H: main('stair'), B: main('brick'), '?': main('question'),
    '[': pipe, ']': pipe, '{': pipe, '}': pipe, o: main('coin'), F: '#dddddd', T: '#f5c842',
  };
  return miniColours.cache;
}

function status() {
  $('st-pos').textContent = S.hover ? t('st.pos', { x: S.hover[0], y: S.hover[1] }) : t('st.posNone');
  $('st-zoom').textContent = t('st.zoom', { z: Math.round(S.view.scale / ART * 100) });
  $('st-width').textContent = t('st.width', { w: S.lv.width });
  $('st-trace').hidden = !S.trace;
  if (S.trace) $('btn-trace').textContent = S.trace.hidden ? t('st.traceShow') : t('st.traceHide');
  const where = savePath();
  $('st-where').textContent = where ? t('st.where', { p: where }) : '';
  $('st-where').title = where;
}

/** The file a save would write, for the status bar. */
function savePath() {
  if (!S.info) return '';
  const src = targetSource();
  const dir = src === 'local' ? S.info.localDir : S.info.builtinDir;
  if (!dir) return '';
  const id = $('level-id').value.trim() || S.id || '…';
  return dir + (dir.includes('\\') ? '\\' : '/') + id + '.json';
}

/** Where a save goes: back to the folder the level came from, unless that
 * folder is read-only (the levels compiled into the program), then the local
 * folder (spec level-editor / 保存位置). */
function targetSource() {
  const src = S.source || S.info?.newSource || 'local';
  return S.info?.writable?.[src] ? src : 'local';
}

// ================================================================ pointer

function normRect(a, b) {
  return { x0: Math.min(a[0], b[0]), y0: Math.min(a[1], b[1]), x1: Math.max(a[0], b[0]), y1: Math.max(a[1], b[1]) };
}

const inRect = (r, [x, y]) => r && x >= r.x0 && x <= r.x1 && y >= r.y0 && y <= r.y1;

let spaceHeld = false;

function onPointerDown(ev) {
  if (S.playing) return;
  canvas.setPointerCapture(ev.pointerId);
  const cell = cellAt(ev);
  // Panning: middle button, or space held.
  if (ev.button === 1 || (ev.button === 0 && spaceHeld)) {
    S.drag = { kind: 'pan', x: ev.clientX, y: ev.clientY, vx: S.view.x, vy: S.view.y };
    canvas.classList.add('panning');
    return;
  }
  // Right button erases, whatever the tool (spec level-editor / 右键擦除).
  const erase = ev.button === 2 || S.tool === 'erase';

  if (S.pastePreview && S.clipboard && ev.button === 0) {
    edit(t('help.clip'), p => M.paste(S.lv, p, S.clipboard, cell[0], cell[1]));
    S.pastePreview = false;
    return;
  }

  // Dragging the spawn or the flag moves it, from any tool but erase.
  if (!erase && S.tool !== 'stamp') {
    if (cell[0] === S.lv.spawn.x && cell[1] === S.lv.spawn.y) {
      S.drag = { kind: 'spawn', patch: M.newPatch(t('el.spawn')) };
      selectThing({ type: 'spawn', x: cell[0], y: cell[1] });
      return;
    }
    if (cell[0] === S.lv.flag.x && cell[1] >= S.lv.flag.topY && cell[1] <= S.lv.flag.bottomY) {
      S.drag = { kind: 'flag', patch: M.newPatch(t('el.flag')) };
      selectThing({ type: 'flag', x: cell[0], y: cell[1] });
      return;
    }
  }

  switch (erase ? 'erase' : S.tool) {
    case 'pen': {
      const p = M.newPatch(t('tool.pen') + ' · ' + t('el.' + S.element));
      S.drag = { kind: 'pen', patch: p, last: cell };
      M.placeElement(S.lv, p, M.elementById(S.element), cell[0], cell[1], S.content);
      draw();
      break;
    }
    case 'erase': {
      if (ev.button === 0 && S.tool === 'erase' && ev.shiftKey) {
        S.drag = { kind: 'rect', start: cell, cur: cell, erase: true };
        break;
      }
      const p = M.newPatch(t('tool.erase'));
      S.drag = { kind: 'erase', patch: p, last: cell };
      M.eraseCell(S.lv, p, cell[0], cell[1]);
      draw();
      break;
    }
    case 'rect':
      S.drag = { kind: 'rect', start: cell, cur: cell, erase: false };
      break;
    case 'pick':
      pick(cell);
      break;
    case 'hand':
      // Selecting without drawing: click anything to edit it on the right.
      selectAt(cell);
      break;
    case 'stamp':
      edit(t('stamp.' + S.stamp),
        p => M.placeStamp(S.lv, p, S.stamp, S.stampSize[S.stamp], cell[0], cell[1]));
      break;
    case 'select':
      if (S.selection && inRect(S.selection, cell)) {
        S.drag = { kind: 'move', start: cell, cur: cell };
      } else {
        S.selection = null;
        S.drag = { kind: 'select', start: cell, cur: cell };
      }
      break;
  }
}

function onPointerMove(ev) {
  const cell = cellAt(ev);
  const changed = !S.hover || S.hover[0] !== cell[0] || S.hover[1] !== cell[1];
  S.hover = cell;
  const d = S.drag;
  if (d?.kind === 'pan') {
    const r = canvas.getBoundingClientRect();
    const scale = (canvas.width / r.width) / S.view.scale;
    S.view.x = clampView(d.vx - (ev.clientX - d.x) * scale);
    S.view.y = clampViewY(d.vy - (ev.clientY - d.y) * scale);
    draw();
    return;
  }
  if (!changed) return;
  if (d?.kind === 'pen' || d?.kind === 'erase') {
    // Every cell between the last and this one, so a fast drag skips none.
    for (const [x, y] of M.line(d.last[0], d.last[1], cell[0], cell[1])) {
      if (d.kind === 'pen') M.placeElement(S.lv, d.patch, M.elementById(S.element), x, y, S.content);
      else M.eraseCell(S.lv, d.patch, x, y);
    }
    d.last = cell;
  } else if (d?.kind === 'rect' || d?.kind === 'select' || d?.kind === 'move') {
    d.cur = cell;
  } else if (d?.kind === 'spawn') {
    M.changeField(S.lv, d.patch, 'spawn', s => { s.x = cell[0]; s.y = cell[1]; });
    S.selected = { type: 'spawn', x: cell[0], y: cell[1] };
  } else if (d?.kind === 'flag') {
    M.moveFlag(S.lv, d.patch, cell[0]);
    S.selected = { type: 'flag', x: cell[0], y: S.lv.flag.topY };
  }
  draw();
}

function onPointerUp() {
  const d = S.drag; S.drag = null;
  canvas.classList.remove('panning');
  if (!d) return;
  switch (d.kind) {
    case 'pen': case 'erase': case 'spawn': case 'flag':
      commit(d.patch);
      if (d.kind === 'pen' && M.elementById(S.element).glyph === '?') selectAt(d.last);
      break;
    case 'rect': {
      const r = normRect(d.start, d.cur);
      edit(d.erase ? t('tool.erase') : t('tool.rect'), p => {
        for (const [x, y] of M.rect(r.x0, r.y0, r.x1, r.y1)) {
          if (d.erase) M.eraseCell(S.lv, p, x, y);
          else M.placeElement(S.lv, p, M.elementById(S.element), x, y, S.content);
        }
      });
      break;
    }
    case 'select': {
      const r = normRect(d.start, d.cur);
      if (r.x0 === r.x1 && r.y0 === r.y1) { S.selection = null; selectAt(d.start); }
      else { S.selection = r; S.selected = null; syncProps(); }
      break;
    }
    case 'move': {
      const dx = d.cur[0] - d.start[0], dy = d.cur[1] - d.start[1];
      if (dx || dy) {
        const r = S.selection;
        edit(t('tool.select'), p => M.moveRect(S.lv, p, r, dx, dy));
        S.selection = { x0: r.x0 + dx, y0: r.y0 + dy, x1: r.x1 + dx, y1: r.y1 + dy };
      }
      break;
    }
  }
  draw();
}

function onWheel(ev) {
  ev.preventDefault();
  const r = canvas.getBoundingClientRect();
  const px = (ev.clientX - r.left) * (canvas.width / r.width);
  const py = (ev.clientY - r.top) * (canvas.height / r.height);
  if (ev.ctrlKey || ev.metaKey) {
    zoomAt(ev.deltaY < 0 ? 1.15 : 1 / 1.15, px, py);
  } else {
    // Wheel scrolls sideways (the level is long and short); shift or a
    // horizontal wheel too.
    const d = (Math.abs(ev.deltaX) > Math.abs(ev.deltaY) ? ev.deltaX : ev.deltaY) / S.view.scale;
    if (ev.shiftKey && Math.abs(ev.deltaX) < Math.abs(ev.deltaY)) S.view.y = clampViewY(S.view.y + d);
    else S.view.x = clampView(S.view.x + d);
    draw();
  }
}

function onMiniPointer(ev) {
  if (ev.buttons !== 1 && ev.type !== 'pointerdown') return;
  const m = mini._map; if (!m) return;
  const r = mini.getBoundingClientRect();
  const x = ((ev.clientX - r.left) * (mini.width / r.width) - m.ox) / m.k;
  S.view.x = clampView(x - canvas.width / S.view.scale / 2);
  draw();
}

// ================================================================ select & properties

function pick(cell) {
  const [x, y] = cell;
  const e = S.lv.entities.find(e => e.x === x && e.y === y);
  const c = S.lv.contents.get(M.key(x, y));
  if (e) setElement(e.kind);
  else if (c && c.hidden) { setElement('hidden'); S.content = structuredClone(c); }
  else {
    const ch = M.at(S.lv, x, y);
    const el = M.ELEMENTS.find(el => el.glyph === ch);
    if (el) setElement(el.id);
    if (c) S.content = structuredClone(c);
  }
  setTool('pen');
  toast(t('toast.pick', { x: t('el.' + S.element) + (c ? ' · ' + kindLabel(c.kind) : '') }), 1500);
}

function selectAt([x, y]) {
  const lv = S.lv;
  if (x === lv.spawn.x && y === lv.spawn.y) return selectThing({ type: 'spawn', x, y });
  if (x === lv.flag.x && y >= lv.flag.topY && y <= lv.flag.bottomY) return selectThing({ type: 'flag', x, y });
  if (lv.contents.has(M.key(x, y))) return selectThing({ type: 'block', x, y });
  selectThing(null);
}

function selectThing(thing) { S.selected = thing; syncProps(); draw(); }

const kindLabel = k => t('kind.' + k);
const kindSprite = k => k === 'coin' ? 'coin' : 'item-' + k;

/** Blocks the property panel is editing: the selected block, or every block
 * in the selection rectangle (spec random-block-content / 批量设置). */
function editedBlocks() {
  if (S.selected?.type === 'block') return [[S.selected.x, S.selected.y]];
  if (S.selection) return M.blocksIn(S.lv, S.selection);
  return [];
}

/** Drop a selection that no longer points at anything — after an undo removed
 * the block, say. */
function pruneSelection() {
  const sel = S.selected;
  if (sel?.type === 'block' && !S.lv.contents.has(M.key(sel.x, sel.y))) S.selected = null;
  if (sel?.type === 'spawn') S.selected = { type: 'spawn', x: S.lv.spawn.x, y: S.lv.spawn.y };
  if (sel?.type === 'flag') S.selected = { type: 'flag', x: S.lv.flag.x, y: S.lv.flag.topY };
}

function syncProps() {
  pruneSelection();
  const box = $('props'); box.innerHTML = '';
  const sel = S.selected;

  // The panel shows what the current tool and element are about.
  if (S.tool === 'stamp') { stampProps(box); return; }

  if (sel?.type === 'spawn') {
    box.innerHTML = `<div class="row"><b>${t('props.spawn')}</b><span class="muted">${t('props.at', { x: S.lv.spawn.x, y: S.lv.spawn.y })}</span></div><p class="muted">${t('props.spawnHint')}</p>`;
    return;
  }
  if (sel?.type === 'flag') {
    const f = S.lv.flag;
    box.innerHTML = `<div class="row"><b>${t('props.flag')}</b><span class="muted">${t('props.col', { x: f.x })}</span></div>
      <label class="row">${t('props.flagHeight')} <input type="number" id="flag-h" min="2" max="${f.bottomY + 1}" value="${f.bottomY - f.topY + 1}"> ${t('props.flagUnit')}</label>
      <p class="muted">${t('props.flagHint')}</p>`;
    $('flag-h').onchange = e => edit(t('props.flagHeight'), p => M.setFlagTop(S.lv, p, f.bottomY - (+e.target.value) + 1));
    return;
  }

  const cells = editedBlocks();
  if (cells.length) { blockProps(box, cells); return; }

  // Nothing selected: with the pen on a block-carrying element the panel edits
  // what the next block will hold. Choosing an element drops any older
  // selection (setElement), so the panel always follows the pen.
  const el = M.elementById(S.element);
  if (S.tool === 'pen' && el) {
    if (el.glyph === M.T.QUESTION || el.hidden) {
      penElementProps(box, el);
      return;
    }
    box.append(elHead(t('el.' + el.id), t('el.' + el.id + '.hint')));
    return;
  }

  box.innerHTML = `<p class="muted">${S.selection ? t('props.noBlocks') : t('props.empty')}</p>`;
}

/** The pen's element: what it is, and — for a question or hidden block — what
 * the next one will hold. Edits go to S.content, the pen's own draft, never to
 * a block already in the level. */
function penElementProps(box, el) {
  box.append(elHead(t('el.' + el.id), t('el.' + el.id + '.hint')));
  const lead = document.createElement('p');
  lead.className = 'muted'; lead.textContent = t('props.penBlock');
  box.append(lead);
  // The draft carries the element's own kind of block: a hidden block stays
  // hidden, a question block does not.
  const draft = { ...S.content, hidden: !!el.hidden };
  contentPanel(box, [draft], {
    apply: content => { S.content = { ...content, hidden: !!el.hidden }; syncProps(); },
    titleText: '',
    pen: true,
  });
}

/** A panel heading: the thing's name over its one-line description. */
function elHead(title, hint) {
  const d = document.createElement('div');
  d.innerHTML = `<div class="row"><b></b></div><p class="muted"></p>`;
  d.children[0].textContent = title;
  d.children[1].textContent = hint;
  return d;
}

/** The stamps, and the size of the current one: the stamp tool's own panel. */
function stampProps(box) {
  const head = elHead(t('props.stamp', { name: t('stamp.' + S.stamp) }), t('props.stampHint'));
  box.append(head);
  const grid = document.createElement('div'); grid.className = 'stamps';
  for (const s of M.STAMPS) {
    const b = document.createElement('button'); b.className = 'stamp'; b.dataset.stamp = s.id;
    b.textContent = t('stamp.' + s.id);
    b.classList.toggle('on', s.id === S.stamp);
    b.onclick = () => { S.stamp = s.id; syncProps(); draw(); };
    grid.append(b);
  }
  box.append(grid);
  const s = M.STAMPS.find(s => s.id === S.stamp);
  const row = document.createElement('label');
  row.className = 'row';
  row.innerHTML = `<span>${t('props.size')}</span><input type="number" id="stamp-size-n" min="${s.min}" max="${s.max}" value="${S.stampSize[s.id]}"><span class="muted">${t('unit.' + s.id)}</span>`;
  box.append(row);
  $('stamp-size-n').onchange = e => {
    S.stampSize[s.id] = Math.max(s.min, Math.min(s.max, +e.target.value || s.min));
    draw(); syncProps();
  };
}

/** The shared contents editor for one or more blocks. `blocks` are cells whose
 * contents are read from the level; for the pen's pending contents it is a
 * single draft object and apply writes it back to S.content. */
function blockProps(box, cells) {
  const first = S.lv.contents.get(M.key(...cells[0]));
  const title = cells.length > 1 ? t('props.nBlocks', { n: cells.length }) : (first.hidden ? t('props.hiddenBlock') : M.at(S.lv, ...cells[0]) === 'B' ? t('props.brick') : t('props.question'));
  box.innerHTML = `<div class="row"><b>${title}</b>${cells.length === 1 ? `<span class="muted">${t('props.at', { x: cells[0][0], y: cells[0][1] })}</span>` : ''}</div>`;
  contentPanel(box, cells.map(([x, y]) => S.lv.contents.get(M.key(x, y))), {
    titleText: '',
    apply: content => edit(t('props.contents'), p => M.setContents(S.lv, p, cells, content)),
  });
}

/** Contents controls: fixed kind or a random pool, times, hidden. `contents`
 * is the current content of the thing being edited (one per block); `apply`
 * commits a new content object. */
function contentPanel(box, contents, { apply, titleText, pen = false }) {
  const first = contents[0];
  const random = first.kind === 'random';

  const head = document.createElement('div');
  head.className = 'row';
  head.innerHTML = `<div class="row"><span>${t('props.contents')}</span><span class="seg"><button id="mode-fixed">${t('props.fixed')}</button><button id="mode-random">${t('props.random')}</button></span></div>
    <div id="kind-area"></div>
    <label class="row">${t('props.count')} <input type="number" id="c-count" min="1" max="99" value="${first.count || 1}"> <span class="muted">${t('props.countHint')}</span></label>
    ${pen ? '' : `<label class="row"><input type="checkbox" id="c-hidden"> ${t('props.hidden')} <span class="muted">${t('props.hiddenHint')}</span></label>`}`;
  box.append(head);

  $('mode-fixed').onclick = () => {
    if (!random) return;
    apply({ kind: 'coin', count: first.count, pool: null });
    if (S.tool === 'pen') syncProps();
  };
  $('mode-random').onclick = () => {
    if (random) return;
    apply({ kind: 'random', count: first.count, pool: M.BLOCK_KINDS.map(k => ({ kind: k, w: 1 })) });
    if (S.tool === 'pen') syncProps();
  };
  $('mode-fixed').classList.toggle('on', !random);
  $('mode-random').classList.toggle('on', random);

  // The pen's panel has no hidden checkbox: the element already decides that.
  const hiddenBox = $('c-hidden');
  if (hiddenBox) {
    hiddenBox.checked = contents.every(c => c.hidden);
    hiddenBox.indeterminate = !hiddenBox.checked && contents.some(c => c.hidden);
    // A hidden block sits in open air; a visible one is a question block.
    hiddenBox.onchange = () => {
      const want = hiddenBox.checked;
      const cells = editedBlocks();
      edit(t('props.hidden'), p => {
        for (const [x, y] of cells) {
          const c = S.lv.contents.get(M.key(x, y));
          M.setTile(S.lv, p, x, y, want ? M.T.AIR : M.T.QUESTION);
          M.setContent(S.lv, p, x, y, { ...c, hidden: want });
        }
      });
    };
  }

  $('c-count').onchange = e => {
    const n = Math.max(1, Math.min(99, +e.target.value || 1));
    if (S.tool === 'pen') { apply({ ...S.content, count: n }); syncProps(); return; }
    const cells = editedBlocks();
    edit(t('props.count'), p => { for (const [x, y] of cells) { const c = S.lv.contents.get(M.key(x, y)); M.setContent(S.lv, p, x, y, { ...c, count: n }); } });
  };

  const area = $('kind-area');
  if (!random) {
    const pickRow = document.createElement('div'); pickRow.className = 'kind-pick';
    for (const k of M.BLOCK_KINDS) {
      const b = document.createElement('button');
      b.append(iconCanvas(kindSprite(k), 20), kindLabel(k));
      b.classList.toggle('on', first.kind === k && contents.length === 1);
      b.onclick = () => {
        apply({ kind: k, count: first.count, pool: null });
        // Choosing here also sets what the pen draws next.
        S.content = { kind: k, count: S.content?.count || 1, pool: null, hidden: !!S.content?.hidden };
        if (S.tool === 'pen') syncProps();
      };
      pickRow.append(b);
    }
    area.append(pickRow);
    return;
  }

  // Random: tick the candidates, give each a weight, see the chances.
  const pool = M.BLOCK_KINDS.map(k => {
    const e = (first.pool || []).find(p => p.kind === k);
    return { kind: k, on: !!e, w: e ? e.w : 1 };
  });
  const list = document.createElement('div'); list.className = 'kinds';
  const pcts = M.probabilities(pool.map(p => ({ kind: p.kind, w: p.on ? p.w : 0 })));
  pool.forEach((p, i) => {
    const row = document.createElement('div'); row.className = 'kind';
    row.innerHTML = `<input type="checkbox" ${p.on ? 'checked' : ''}><span></span><span>${kindLabel(p.kind)}</span>
      <input type="number" min="0" max="999" value="${p.w}" ${p.on ? '' : 'disabled'} title="${t('props.weight')}"><span class="pct">${p.on ? pcts[i] + '%' : '–'}</span>`;
    row.children[1].replaceWith(iconCanvas(kindSprite(p.kind), 24));
    const [chk, , , weight] = row.querySelectorAll('input, canvas, span');
    const save = () => {
      const next = pool.map((q, j) => j === i ? { ...q, on: chk.checked, w: Math.max(0, +weight.value || 0) } : q)
        .filter(q => q.on).map(q => ({ kind: q.kind, w: q.w }));
      if (!next.length) { toast(t('props.keepOne'), 2000, true); syncProps(); return; }
      apply({ kind: 'random', count: first.count, pool: next });
      if (S.tool === 'pen') syncProps();
    };
    row.querySelector('input[type=checkbox]').onchange = save;
    row.querySelector('input[type=number]').onchange = save;
    list.append(row);
  });
  area.append(list);
  const total = pool.filter(p => p.on).reduce((s, p) => s + p.w, 0);
  if (total === 0) {
    const warn = document.createElement('p'); warn.className = 'muted';
    warn.style.color = 'var(--error)'; warn.textContent = t('props.zeroWeights');
    area.append(warn);
  }
  const note = document.createElement('p'); note.className = 'muted';
  note.textContent = t('props.drawNote');
  area.append(note);
  if (titleText) box.insertBefore(Object.assign(document.createElement('div'), { className: 'row', textContent: titleText }), head);
}

// ================================================================ level panel

function syncLevelPanel() {
  const lv = S.lv;
  setVal('level-title', lv.meta.title || '');
  setVal('level-author', lv.meta.author || '');
  setVal('level-desc', lv.meta.description || '');
  setVal('level-diff', String(lv.meta.difficulty || 0));
  setVal('level-time', lv.timeLimit);
  setVal('level-width', lv.width);
  setVal('level-id', S.id);
  const src = targetSource();
  $('level-where').textContent = (src === 'local' ? t('lvl.whereLocal') : t('lvl.whereBuiltin')) +
    (S.openedAs ? '' : ' · ' + t('lvl.whereNew'));
  $('level-where').title = savePath();
  document.title = (lv.meta.title || S.id || t('untitled')) + ' · ' + t('app.title');
}

function setVal(id, v) { const el = $(id); if (document.activeElement !== el) el.value = v; }

function metaChange(field, value) {
  if ((S.lv.meta[field] || '') === value) return;
  edit(t('sec.level'), p => M.changeField(S.lv, p, 'meta', m => { if (value) m[field] = value; else delete m[field]; }));
}

// ================================================================ appearance

function buildLook() {
  const themes = $('themes');
  for (const th of S.palettes.themes) {
    const name = palName(th);
    const b = document.createElement('button'); b.className = 'theme-card'; b.dataset.theme = th.ID;
    const c = document.createElement('canvas');
    const f = mario.themePreview(th.ID, 4);
    c.width = f.w; c.height = f.h;
    c.getContext('2d').putImageData(new ImageData(f.rgba, f.w, f.h), 0, 0);
    const label = document.createElement('span'); label.textContent = name;
    b.append(c, label);
    b.onclick = () => setStyle(t('sec.look') + ' · ' + name, () => ({ theme: th.ID }));
    themes.append(b);
    // The same cards in the new-level dialog.
    const nb = b.cloneNode(true);
    nb.querySelector('canvas').getContext('2d').drawImage(c, 0, 0);
    nb.onclick = () => { document.querySelectorAll('#new-themes .theme-card').forEach(x => x.classList.remove('on')); nb.classList.add('on'); };
    $('new-themes').append(nb);
  }
  $('new-themes').firstChild?.classList.add('on');

  const rows = $('look-rows');
  rows.append(skyRow());
  for (const field of ['turf', 'ground', 'brick', 'stair', 'pipe']) {
    rows.append(colourRow(field, t('look.' + field), S.palettes[field]));
  }
  $('btn-reset-look').onclick = () => setStyle(t('look.reset'), s => (s.theme ? { theme: s.theme } : {}));
}

function skyRow() {
  const row = document.createElement('div'); row.className = 'look-row'; row.dataset.field = 'sky';
  row.innerHTML = `<div class="lbl"><span>${t('look.sky')}</span><span class="sky-mode">
      <label><input type="radio" name="sky-mode" value="solid"> ${t('look.solid')}</label>
      <label><input type="radio" name="sky-mode" value="grad"> ${t('look.grad')}</label></span></div>
    <div class="swatches"></div>
    <div class="swatches custom">${t('look.custom')} <input type="color" id="sky-top" title="${t('look.skyTop.tip')}"> <input type="color" id="sky-bottom" title="${t('look.skyBottom.tip')}"></div>`;
  const sw = row.querySelector('.swatches');
  for (const s of S.palettes.skies) {
    const b = document.createElement('button');
    b.className = 'swatch grad'; b.title = palName(s); b.dataset.value = s.ID;
    b.style.background = `linear-gradient(${s.Top}, ${s.Bottom})`;
    b.onclick = () => setStyle(t('look.sky') + ' · ' + palName(s), st => ({ ...st, sky: s.ID }));
    sw.append(b);
  }
  row.querySelectorAll('input[name=sky-mode]').forEach(r => r.onchange = () => {
    const top = $('sky-top').value, bottom = $('sky-bottom').value;
    setStyle(t('look.sky'), st => ({ ...st, sky: r.value === 'grad' ? { top, bottom } : top }));
  });
  const custom = () => {
    const grad = row.querySelector('input[value=grad]').checked;
    const top = $('sky-top').value, bottom = $('sky-bottom').value;
    setStyle(t('look.sky'), st => ({ ...st, sky: grad ? { top, bottom } : top }));
  };
  row.querySelector('#sky-top').onchange = custom;
  row.querySelector('#sky-bottom').onchange = custom;
  return row;
}

function colourRow(field, label, list) {
  const row = document.createElement('div'); row.className = 'look-row'; row.dataset.field = field;
  row.innerHTML = `<div class="lbl"><span>${label}</span></div><div class="swatches"></div>`;
  const sw = row.querySelector('.swatches');
  for (const s of list) {
    const b = document.createElement('button');
    b.className = 'swatch'; b.title = palName(s); b.dataset.value = s.ID; b.style.background = s.Hex;
    b.onclick = () => setStyle(label + ' · ' + palName(s), st => ({ ...st, [field]: s.ID }));
    sw.append(b);
  }
  const pickr = document.createElement('input'); pickr.type = 'color'; pickr.title = t('look.customOf', { x: label });
  pickr.onchange = () => setStyle(label, st => ({ ...st, [field]: pickr.value }));
  sw.append(pickr);
  return row;
}

/** A palette entry's name in the current language. */
const palName = p => (currentLang() === 'en' ? p.LabelEn : p.Label) || p.ID;

/** Change the style as one undoable step; the atlas and sky follow. */
function setStyle(label, fn) {
  edit(label, p => M.changeField(S.lv, p, 'style', st => {
    const next = fn(structuredClone(st));
    for (const k of Object.keys(st)) delete st[k];
    Object.assign(st, next);
  }));
}

/** Show which theme and swatches are in use. */
function syncLook() {
  const st = S.lv.style || {};
  document.querySelectorAll('#themes .theme-card').forEach(b => b.classList.toggle('on', b.dataset.theme === st.theme));
  document.querySelectorAll('#look-rows .look-row').forEach(row => {
    const v = st[row.dataset.field];
    row.querySelectorAll('.swatch').forEach(b => b.classList.toggle('on', typeof v === 'string' && b.dataset.value === v));
  });
  const sky = st.sky;
  const grad = sky && typeof sky === 'object';
  const r = document.querySelector(`input[name=sky-mode][value=${grad ? 'grad' : 'solid'}]`);
  if (r) r.checked = true;
  $('sky-bottom').hidden = !grad;
  const top = S.sky[0] || '#000000', bottom = S.sky.at(-1) || top;
  $('sky-top').value = top; $('sky-bottom').value = bottom;
}

// ================================================================ save

async function save() {
  if (S.saving || S.opening) return;
  const errs = errors();
  if (errs.length) {
    toast(t('toast.saveErrors', { n: errs.length, m: issueText(errs[0]) }), 5000, true);
    focusCell(errs[0].x, errs[0].y);
    return;
  }
  let id = $('level-id').value.trim() || S.id;
  if (!id) {
    id = M.suggestId(S.lv.meta.author, S.lv.meta.title);
    const answer = prompt(t('q.askId'), id);
    if (!answer) return;
    id = answer.trim();
  }
  if (!M.validId(id)) {
    toast(t(M.validId.reason(id)), 5000, true);
    $('level-id').focus(); return;
  }

  const out = JSON.parse(mario.format(M.toFile({ ...S.lv, name: id, world: S.lv.world || id })));
  if (out.error) { toast(out.error, 5000, true); return; }

  // Back where it came from; a read-only built-in level goes to the local
  // folder instead, and says so.
  const wanted = S.source || S.info.newSource;
  const source = targetSource();
  const { revision, openedAs, baseRevision } = S;
  const put = (overwrite, revision) => fetch('api/levels/' + source + '/' + id, {
    method: 'PUT', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ level: out.text, overwrite, openedAs, revision }),
  });
  S.saving = true;
  $('btn-save').disabled = true;
  try {
    let r = await put(false, baseRevision);
    let body = await r.json();
    if (r.status === 409 && body.code === 'exists') {
      const where = source === 'local' ? t('lvl.whereLocal') : t('lvl.whereBuiltin');
      if (!(await confirmDialog(t('q.overwriteTitle'), t('q.overwrite', { id, where })))) return;
      r = await put(true, body.revision);
      body = await r.json();
    }
    if (!r.ok) { toast(t('err.' + body.code, body) || t('err.unknown'), 6000, true); return; }
    const msg = wanted !== source ? t('toast.savedReadOnly', { p: body.path }) : t('toast.saved', { p: body.path });
    finishSave(id, source, body.revision, revision, msg);
  } catch {
    toast(t('err.save-request'), 6000, true);
  } finally {
    S.saving = false;
    $('btn-save').disabled = false;
  }
}

function finishSave(id, source, baseRevision, revision, msg) {
  S.source = source; S.openedAs = keyOf(source, id); S.baseRevision = baseRevision;
  S.lv.name = id;
  if (!S.lv.world) S.lv.world = id;
  if (S.revision === revision) {
    S.id = id;
    setDirty(false);
    clearDraft();
  } else {
    flushDraft();
    msg = t('toast.savedWithPendingEdits');
  }
  history.replaceState(null, '', '#' + S.openedAs);
  syncLevelPanel();
  toast(msg, 4000);
  draw();
}

// ================================================================ drafts

let draftTimer = 0;
function saveDraftSoon() {
  clearTimeout(draftTimer);
  draftTimer = setTimeout(flushDraft, 350);
}

function flushDraft() {
  clearTimeout(draftTimer);
  draftTimer = 0;
  if (!S.dirty) return;
  const { id, source, openedAs, baseRevision, draftToken, revision } = S;
  const at = Date.now();
  storeDraft({ token: draftToken, id, source, openedAs, revision: baseRevision, level: M.toFile(S.lv), at }).then(() => {
    if (S.dirty && S.revision === revision) $('st-draft').textContent = t('st.draft', { t: new Date(at).toLocaleTimeString().slice(0, 5) });
  }, () => {
    if (S.dirty && S.revision === revision) $('st-draft').textContent = t('st.draftFail');
  });
}

async function restoreDraft(key) {
  const draft = await readDraft(key);
  if (!draft || draft.token === S.draftToken) return false;
  if (!confirmSync(t('q.restore', { id: draft.id || t('untitled'), at: new Date(draft.at).toLocaleString() }))) return false;
  try {
    const lv = M.fromFile(draft.level);
    clearDraft();
    loadDoc(lv, draft.id, draft.source || S.info.newSource, draft.openedAs || '', true, draft.revision || '', draft.token);
    if (S.openedAs) history.replaceState(null, '', '#' + S.openedAs);
    return true;
  } catch {
    toast(t('toast.draftOpenFailed'), 5000, true);
    return false;
  }
}

function clearDraft() {
  clearTimeout(draftTimer);
  draftTimer = 0;
  if (S.draftToken) deleteDraft(S.draftToken).catch(() => {});
  $('st-draft').textContent = '';
}

// ================================================================ playtest

const KEYS = { ArrowLeft: 1, ArrowRight: 2, ArrowDown: 4, ArrowUp: 8, Space: 8, KeyZ: 8, ShiftLeft: 16, ShiftRight: 16, KeyX: 16 };

function startPlay(fromHere) {
  if (S.playing) return;
  const errs = errors();
  if (errs.length) {
    toast(t('toast.playErrors', { m: issueText(errs[0]) }), 5000, true);
    focusCell(errs[0].x, errs[0].y);
    return;
  }
  const o = S.playOpts;
  if (!o.fixedSeed) o.seed = 1 + Math.floor(Math.random() * 2147483646);
  const viewW = Math.ceil(canvas.width / S.view.scale), viewH = M.HEIGHT;
  const opts = {
    fromColumn: fromHere && S.hover ? S.hover[0] : -1,
    startBig: o.startBig, invincible: o.invincible, seed: o.seed, showDraws: o.showDraws,
    zoom: ART, viewW: Math.max(16, viewW), viewH,
  };
  const res = JSON.parse(mario.start(M.toFile(S.lv), JSON.stringify(opts)));
  if (res.issues) { toast(t('toast.noPlay', { m: issueText(res.issues[0]) }), 5000, true); return; }
  if (res.notice) toast(t('toast.noFoothold'), 3000);

  S.playing = { keys: 0, last: performance.now(), acc: 0, raf: 0, steps: 0 };
  $('play-canvas').hidden = false; $('play-hud').hidden = false;
  canvas.style.visibility = 'hidden';
  document.activeElement?.blur();
  S.playing.raf = requestAnimationFrame(playFrame);
}

/** One browser frame: step the game at 60 Hz whatever the frame rate, then
 * draw it (design decision 6). */
function playFrame(now) {
  const P = S.playing; if (!P) return;
  P.acc += Math.min(100, now - P.last); P.last = now;
  const steps = Math.floor(P.acc / (1000 / 60));
  P.acc -= steps * (1000 / 60);
  let end = '';
  if (steps > 0) { end = mario.step(P.keys, Math.min(steps, 5)); P.steps += Math.min(steps, 5); }
  const f = mario.frame();
  if (f) paintPlay(f);
  if (end) { stopPlay(end); return; }
  P.raf = requestAnimationFrame(playFrame);
}

function paintPlay(f) {
  const pc = $('play-canvas'), g = pc.getContext('2d');
  if (!paintPlay.buf || paintPlay.buf.width !== f.w || paintPlay.buf.height !== f.h) {
    paintPlay.buf = document.createElement('canvas');
    paintPlay.buf.width = f.w; paintPlay.buf.height = f.h;
  }
  paintPlay.buf.getContext('2d').putImageData(new ImageData(f.rgba, f.w, f.h), 0, 0);
  g.imageSmoothingEnabled = false;
  g.fillStyle = '#000'; g.fillRect(0, 0, pc.width, pc.height);
  const k = Math.min(pc.width / f.w, pc.height / f.h);
  const w = f.w * k, h = f.h * k;
  const ox = (pc.width - w) / 2, oy = (pc.height - h) / 2;
  g.drawImage(paintPlay.buf, ox, oy, w, h);
  // Random draws, when asked for: the item over its untouched block.
  const draws = JSON.parse(f.draws || 'null');
  if (draws) {
    const tile = h / M.HEIGHT;
    for (const d of draws) {
      const sp = S.sprites.get(kindSprite(d.Kind)); if (!sp) continue;
      const px = ox + (d.X - f.cameraX) * tile, py = oy + (d.Y - f.cameraY - 0.8) * tile;
      g.globalAlpha = .9; g.drawImage(sp.canvas, px + tile * .2, py, tile * .6, tile * .6); g.globalAlpha = 1;
    }
  }
}

function stopPlay(end) {
  const P = S.playing; if (!P) return;
  cancelAnimationFrame(P.raf);
  S.playing = null;
  const sum = JSON.parse(mario.stop() || 'null');
  $('play-canvas').hidden = true; $('play-hud').hidden = true;
  canvas.style.visibility = '';
  if (sum) {
    S.lastRun = { end: sum.end, column: sum.column, steps: sum.steps };
    S.trace = { points: sum.trace || [], death: sum.death || null, stale: false, hidden: false };
    // Put the view where the run ended (spec level-playtest / 回到编辑).
    S.view.x = clampView(sum.column - canvas.width / S.view.scale / 2);
    const msg = end === 'cleared' ? t('toast.cleared') : end === 'died' ? t('toast.died', { x: Math.floor(sum.column) }) : '';
    if (msg) toast(msg, 2500);
  }
  draw();
}

// ================================================================ keyboard

const SHORTCUTS = [
  ['B', 'help.pen'], ['R', 'help.rect'], ['E', 'help.erase'], ['I', 'help.pick'], ['H', 'help.hand'],
  ['V', 'help.select'], ['S', 'help.stamp'],
  ['1–9, 0', 'help.elements'], ['[ / ]', 'help.stampSize'], ['F', 'help.fill'], ['T', 'help.look'],
  ['Ctrl+Z', 'help.undo'], ['Ctrl+Shift+Z / Ctrl+Y', 'help.redo'],
  ['Ctrl+C / Ctrl+X / Ctrl+V', 'help.clip'], ['Delete', 'help.delete'], ['Esc', 'help.esc'],
  ['Ctrl+S', 'help.save'], ['Enter / F5', 'help.play'], ['Shift+Enter', 'help.playHere'],
  ['G', 'help.grid'], ['+ / −', 'help.zoom'], ['key.space', 'help.pan'], ['key.right', 'help.rightClick'],
  ['key.wheel', 'help.wheel'], ['?', 'help.this'],
];

function buildHelp() {
  const b = $('help-body');
  for (const [k, what] of SHORTCUTS) {
    const d = document.createElement('div');
    d.innerHTML = `<span></span><span class="k"></span>`;
    d.children[0].textContent = t(what); d.children[1].textContent = k.startsWith('key.') ? t(k) : k;
    b.append(d);
  }
}

function onKeyDown(ev) {
  // While playing, only the game's keys and Esc do anything
  // (spec level-playtest / 编辑快捷键不误触).
  if (S.playing) {
    if (ev.code === 'Escape') { stopPlay(''); ev.preventDefault(); return; }
    const bit = KEYS[ev.code];
    if (bit) { S.playing.keys |= bit; ev.preventDefault(); }
    else if (ev.ctrlKey || ev.metaKey) ev.preventDefault();
    return;
  }
  const tgt = ev.target;
  const typing = tgt instanceof HTMLInputElement && !['checkbox', 'radio', 'color'].includes(tgt.type) || tgt instanceof HTMLTextAreaElement || tgt instanceof HTMLSelectElement;
  if (typing) {
    if (ev.key === 'Escape') tgt.blur();
    if ((ev.ctrlKey || ev.metaKey) && ev.key.toLowerCase() === 's') { ev.preventDefault(); tgt.blur(); save(); }
    return;
  }
  if (document.querySelector('dialog[open]')) return;

  const ctrl = ev.ctrlKey || ev.metaKey;
  const k = ev.key.toLowerCase();
  const handled = () => ev.preventDefault();

  if (ctrl) {
    if (k === 'z' && !ev.shiftKey) { undo(); return handled(); }
    if ((k === 'z' && ev.shiftKey) || k === 'y') { redo(); return handled(); }
    if (k === 's') { save(); return handled(); }
    if (k === 'c' || k === 'x') {
      if (!S.selection) { toast(t('toast.selectFirst'), 2000); return handled(); }
      S.clipboard = M.copyRect(S.lv, S.selection);
      if (k === 'x') edit(t('toast.cut'), p => M.clearRect(S.lv, p, S.selection));
      toast(k === 'x' ? t('toast.cut') : t('toast.copied'), 1000);
      return handled();
    }
    if (k === 'v') { if (S.clipboard) { S.pastePreview = true; draw(); } return handled(); }
    if (k === '=' || k === '+') { zoomAt(1.25, canvas.width / 2, canvas.height / 2); return handled(); }
    if (k === '-') { zoomAt(1 / 1.25, canvas.width / 2, canvas.height / 2); return handled(); }
    return;
  }

  if (ev.code === 'Space') { spaceHeld = true; return handled(); }
  if (ev.key === 'Enter' || ev.key === 'F5') { startPlay(ev.shiftKey); return handled(); }
  if (ev.key === 'Escape') {
    S.pastePreview = false; S.selection = null; S.selected = null; S.drag = null;
    syncProps(); draw(); return;
  }
  if (ev.key === 'Delete' || ev.key === 'Backspace') {
    if (S.selection) edit(t('help.delete'), p => M.clearRect(S.lv, p, S.selection));
    else if (S.selected?.type === 'block') edit(t('help.delete'), p => M.eraseCell(S.lv, p, S.selected.x, S.selected.y));
    return handled();
  }
  if (ev.key === '?') { toggleHelp(); return handled(); }
  const tools = { b: 'pen', r: 'rect', e: 'erase', i: 'pick', h: 'hand', v: 'select', s: 'stamp' };
  if (tools[k]) { setTool(tools[k]); return handled(); }
  const el = M.ELEMENTS.find(e => e.key === ev.key);
  if (el) { setElement(el.id); if (S.tool !== 'rect') setTool('pen'); return handled(); }
  if (k === 'g') { S.grid = !S.grid; draw(); return handled(); }
  if (k === 'f') { fillGround(); return handled(); }
  if (k === 't') { $('look-section').scrollIntoView({ behavior: 'smooth' }); $('look-section').animate([{ background: 'var(--hover)' }, { background: 'transparent' }], 900); return handled(); }
  if (ev.key === '[' || ev.key === ']') { stepStamp(ev.key === ']' ? 1 : -1); return handled(); }
  if (ev.key === '=' || ev.key === '+') { zoomAt(1.25, canvas.width / 2, canvas.height / 2); return handled(); }
  if (ev.key === '-') { zoomAt(1 / 1.25, canvas.width / 2, canvas.height / 2); return handled(); }
  if (ev.key === 'ArrowLeft' || ev.key === 'ArrowRight') {
    S.view.x = clampView(S.view.x + (ev.key === 'ArrowRight' ? 1 : -1) * (ev.shiftKey ? 10 : 2)); draw(); return handled();
  }
}

function onKeyUp(ev) {
  if (ev.code === 'Space') spaceHeld = false;
  if (S.playing) { const bit = KEYS[ev.code]; if (bit) S.playing.keys &= ~bit; }
}

function toggleHelp() { const d = $('dlg-help'); if (d.open) d.close(); else d.showModal(); }

// ================================================================ palettes

const TOOLS = [
  { id: 'hand', key: 'H' }, { id: 'pen', key: 'B' }, { id: 'rect', key: 'R' },
  { id: 'erase', key: 'E' }, { id: 'pick', key: 'I' }, { id: 'select', key: 'V' }, { id: 'stamp', key: 'S' },
];

function buildTools() {
  for (const tl of TOOLS) {
    const b = document.createElement('button'); b.className = 'tool'; b.dataset.tool = tl.id;
    b.innerHTML = `${t('tool.' + tl.id)}<span class="k">${tl.key}</span>`;
    b.title = `${t('tool.' + tl.id)} (${tl.key})`;
    b.onclick = () => setTool(tl.id);
    $('tools').append(b);
  }
  syncTools();
}

function buildElements() {
  for (const e of M.ELEMENTS) {
    const b = document.createElement('button'); b.className = 'element'; b.dataset.element = e.id;
    if (e.sprite) b.append(iconCanvas(e.sprite));
    else { const g = document.createElement('span'); g.className = 'glyphbox'; g.textContent = e.spawn ? '⬆' : '⚑'; b.append(g); }
    const l = document.createElement('span'); l.innerHTML = `${t('el.' + e.id)}<span class="k">${e.key}</span>`;
    b.append(l);
    b.title = `${t('el.' + e.id)} (${e.key})`;
    b.onclick = () => { setElement(e.id); if (S.tool !== 'rect') setTool('pen'); };
    $('elements').append(b);
  }
  syncTools();
}

function buildStamps() {
  // The stamps live in the property panel, beside the tool that uses them.
  syncProps();
}

function stepStamp(d) {
  const s = M.STAMPS.find(s => s.id === S.stamp);
  S.stampSize[s.id] = Math.max(s.min, Math.min(s.max, S.stampSize[s.id] + d));
  syncTools(); draw();
}

function setTool(tool) { S.tool = tool; S.pastePreview = false; syncTools(); syncProps(); draw(); }
/** Choosing an element also drops any block selected earlier: the property
 * panel then follows the pen instead of the old block, which is what the user
 * expects when they switch to another element. */
function setElement(id) {
  if (id !== S.element) S.selected = null;
  S.element = id;
  syncTools(); syncProps();
}

function syncTools() {
  document.querySelectorAll('.tool').forEach(b => b.classList.toggle('on', b.dataset.tool === S.tool));
  document.querySelectorAll('.element').forEach(b => b.classList.toggle('on', b.dataset.element === S.element && S.tool === 'pen'));
  document.querySelectorAll('.stamp').forEach(b => b.classList.toggle('on', S.tool === 'stamp' && b.dataset.stamp === S.stamp));
}

function fillGround() {
  const layers = +$('fill-layers').value, keepGaps = $('fill-keep').checked;
  const r = S.selection;
  edit(t('fill.btn'), p => M.fillGround(S.lv, { layers, keepGaps, x0: r ? r.x0 : 0, x1: r ? r.x1 : S.lv.width - 1 }, p));
  toast(r ? t('toast.filledRange', { a: r.x0, b: r.x1 }) : t('toast.filled'), 1500);
}

// ================================================================ dialogs

async function openDialog() {
  if (S.saving || S.opening) return;
  if (S.dirty && !(await confirmDialog(t('dlg.open'), t('q.discardOpen')))) return;
  const list = await (await fetch('api/levels')).json();
  const ul = $('open-list'); ul.innerHTML = '';
  if (!list.length) {
    const li = document.createElement('li'); li.className = 'muted'; li.textContent = t('open.none'); ul.append(li);
  }
  let group = '';
  for (const l of list) {
    // A heading for each folder, with where it is on disk.
    if (l.source !== group) {
      group = l.source;
      const h = document.createElement('li'); h.className = 'open-group';
      const dir = group === 'local' ? S.info.localDir : S.info.builtinDir;
      h.innerHTML = `<b></b> <span class="sub"></span>`;
      h.querySelector('b').textContent = group === 'local' ? t('open.local') : t('open.builtin');
      h.querySelector('.sub').textContent = dir ? t('open.dir', { p: dir }) : '';
      ul.append(h);
    }
    const li = document.createElement('li');
    const b = document.createElement('button');
    b.dataset.key = keyOf(l.source, l.id);
    b.innerHTML = `<span><b></b><br><span class="sub"></span></span><span class="sub"></span>`;
    b.querySelector('b').textContent = l.title || l.id;
    b.querySelectorAll('.sub')[0].textContent = l.error ? t('open.broken', { e: l.error }) : l.id + (l.author ? ' · ' + l.author : '');
    b.querySelectorAll('.sub')[1].textContent = (l.official ? t('open.official') : t('open.community')) + (l.difficulty ? ' · ' + t('open.diff', { d: l.difficulty }) : '');
    if (l.error) b.disabled = true;
    b.onclick = async () => {
      if (S.saving || S.opening) return;
      $('dlg-open').close();
      const key = keyOf(l.source, l.id);
      S.opening = true;
      try {
        if (!(await restoreDraft(key))) await openLevel(key);
      } finally {
        S.opening = false;
      }
    };
    li.append(b); ul.append(li);
  }
  $('dlg-open').showModal();
}

async function newDialog() {
  if (S.saving || S.opening) return;
  if (S.dirty && !(await confirmDialog(t('dlg.new'), t('q.discardNew')))) return;
  const dir = S.info.newSource === 'local' ? S.info.localDir : S.info.builtinDir;
  $('new-where').textContent = t('dlg.newWhere', { p: dir });
  $('dlg-new').showModal();
}

function newLevel() {
  if (S.saving || S.opening) return;
  const theme = document.querySelector('#new-themes .theme-card.on')?.dataset.theme || 'grassland';
  const width = Math.max(32, Math.min(1000, +$('new-width').value || 120));
  $('dlg-new').close();
  clearDraft();
  loadDoc(M.starter(width, theme), '', S.info.newSource, '', false);
  history.replaceState(null, '', location.pathname);
}

function confirmDialog(title, text) {
  title = title || t('dlg.confirm');
  return new Promise(resolve => {
    const d = $('dlg-confirm');
    $('confirm-title').textContent = title; $('confirm-text').textContent = text;
    const done = v => { d.close(); resolve(v); };
    $('confirm-yes').onclick = () => done(true);
    $('confirm-no').onclick = () => done(false);
    d.onclose = () => resolve(false);
    d.showModal();
  });
}

function confirmSync(text) { return window.confirm(text); }

let toastTimer = 0;
function toast(msg, ms = 2000, error = false) {
  const el = $('toast');
  el.textContent = msg; el.hidden = false; el.classList.toggle('error', error);
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { el.hidden = true; }, ms);
}

// ================================================================ wiring

function wire() {
  canvas.addEventListener('pointerdown', onPointerDown);
  canvas.addEventListener('pointermove', onPointerMove);
  canvas.addEventListener('pointerup', onPointerUp);
  canvas.addEventListener('pointercancel', onPointerUp);
  canvas.addEventListener('pointerleave', () => { if (!S.drag) { S.hover = null; draw(); } });
  canvas.addEventListener('wheel', onWheel, { passive: false });
  canvas.addEventListener('contextmenu', e => e.preventDefault());
  canvas.addEventListener('auxclick', e => e.preventDefault());
  mini.addEventListener('pointerdown', e => { mini.setPointerCapture(e.pointerId); onMiniPointer(e); });
  mini.addEventListener('pointermove', onMiniPointer);
  window.addEventListener('keydown', onKeyDown);
  window.addEventListener('keyup', onKeyUp);
  window.addEventListener('blur', () => { spaceHeld = false; if (S.playing) S.playing.keys = 0; });
  window.addEventListener('resize', resize);
  // Unsaved changes: the browser asks before leaving (spec level-editor / 离开前确认).
  document.addEventListener('visibilitychange', () => {
    if (document.hidden && draftTimer) flushDraft();
  });
  window.addEventListener('beforeunload', e => {
    if (S.dirty) {
      if (draftTimer) flushDraft();
      e.preventDefault();
      e.returnValue = '';
    }
  });

  $('level-id').addEventListener('input', () => status());
  $('btn-new').onclick = newDialog;
  $('btn-new-ok').onclick = newLevel;
  $('btn-open').onclick = openDialog;
  $('btn-save').onclick = save;
  $('btn-undo').onclick = undo;
  $('btn-redo').onclick = redo;
  $('btn-play').onclick = () => startPlay(false);
  $('btn-play-here').onclick = () => {
    if (!S.hover) { toast(t('toast.playHere'), 3000); return; }
    startPlay(true);
  };
  $('btn-help').onclick = toggleHelp;
  $('btn-fill').onclick = fillGround;
  $('btn-trace').onclick = () => { S.trace.hidden = !S.trace.hidden; draw(); };

  const o = S.playOpts;
  $('opt-big').onchange = e => { o.startBig = e.target.checked; };
  $('opt-invincible').onchange = e => { o.invincible = e.target.checked; };
  $('opt-fixed-seed').onchange = e => { o.fixedSeed = e.target.checked; };
  $('opt-seed').onchange = e => { o.seed = Math.max(1, +e.target.value || 1); o.fixedSeed = true; $('opt-fixed-seed').checked = true; };
  $('btn-reseed').onclick = () => { o.seed = 1 + Math.floor(Math.random() * 99999); $('opt-seed').value = o.seed; o.fixedSeed = true; $('opt-fixed-seed').checked = true; toast(t('toast.newSeed', { s: o.seed }), 1200); };
  $('opt-show-draws').onchange = e => { o.showDraws = e.target.checked; };

  $('level-title').onchange = e => metaChange('title', e.target.value.trim());
  $('level-author').onchange = e => metaChange('author', e.target.value.trim());
  $('level-desc').onchange = e => metaChange('description', e.target.value.trim());
  $('level-diff').onchange = e => {
    const v = +e.target.value;
    edit(t('lvl.diff'), p => M.changeField(S.lv, p, 'meta', m => { if (v) m.difficulty = v; else delete m.difficulty; }));
  };
  $('level-time').onchange = e => {
    const v = Math.max(10, Math.min(999, +e.target.value || 400));
    edit(t('lvl.time'), p => M.setValue(S.lv, p, 'timeLimit', v));
  };
  $('level-id').onchange = e => {
    const v = e.target.value.trim();
    if (v && !M.validId(v)) toast(t(M.validId.reason(v)), 5000, true);
    else { S.id = v; setDirty(true); syncLevelPanel(); draw(); }
  };
  $('btn-width').onclick = async () => {
    const w = Math.max(16, Math.min(1000, +$('level-width').value || S.lv.width));
    const side = $('width-side').value;
    const lost = M.lostByResize(S.lv, w, side);
    if (lost.length && !(await confirmDialog(t('q.resizeTitle'), t('q.resize', { n: lost.length, x: lost[0] })))) {
      syncLevelPanel(); return;
    }
    edit(t('lvl.width') + ' ' + w, p => M.resize(S.lv, p, w, side));
  };
}

boot();

// A read-only window into the editor for the browser tests: where a cell is
// on screen, and the level as a file. It changes nothing.
globalThis.__editor = {
  cellCentre(x, y) {
    const r = canvas.getBoundingClientRect(), k = r.width / canvas.width;
    const [px, py] = tileToPx(x + 0.5, y + 0.5);
    return [r.left + px * k, r.top + py * k];
  },
  level: () => M.toFile(S.lv),
  view: () => ({ ...S.view }),
  playing: () => !!S.playing,
  playKeys: () => S.playing ? S.playing.keys : -1,
  playSteps: () => S.playing ? S.playing.steps : -1,
  lastRun: () => S.lastRun || null,
  selected: () => ({ tool: S.tool, selected: S.selected ? { ...S.selected } : null }),
  where: () => ({ source: S.source, target: targetSource(), openedAs: S.openedAs, path: savePath() }),
  /** Scroll so column x is on screen, as a user would before clicking it. */
  reveal(x) {
    const cols = canvas.width / S.view.scale;
    if (x < S.view.x + 1 || x > S.view.x + cols - 2) { S.view.x = clampView(x - cols / 2); paint(); }
  },
};
