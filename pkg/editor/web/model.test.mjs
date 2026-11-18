// model.test.mjs — unit tests for the editor's level model.
// Run with: node --test pkg/editor/web/
import { test } from 'node:test';
import assert from 'node:assert/strict';
import * as M from './model.js';

const grid = lv => {
  const rows = [];
  for (let y = 0; y < M.HEIGHT; y++) rows.push(lv.tiles.slice(y * lv.width, (y + 1) * lv.width).join(''));
  return rows;
};

// ------------------------------------------------------------ 7.2 drawing

test('line: a fast diagonal drag leaves no gaps', () => {
  for (const [x0, y0, x1, y1] of [[0, 0, 30, 0], [0, 0, 7, 3], [5, 9, 0, 2], [0, 0, 3, 11], [2, 2, 2, 2]]) {
    const cells = M.line(x0, y0, x1, y1);
    assert.deepEqual(cells[0], [x0, y0]);
    assert.deepEqual(cells.at(-1), [x1, y1]);
    for (let i = 1; i < cells.length; i++) {
      const d = Math.abs(cells[i][0] - cells[i - 1][0]) + Math.abs(cells[i][1] - cells[i - 1][1]);
      assert.equal(d, 1, `cells ${i - 1} and ${i} of ${x0},${y0}->${x1},${y1} do not share an edge`);
    }
  }
});

test('line: a 30-column drag covers all 30 columns', () => {
  const lv = M.blank(60);
  const p = M.newPatch('pen');
  for (const [x, y] of M.line(5, 12, 34, 12)) M.placeElement(lv, p, M.elementById('ground'), x, y);
  const row = grid(lv)[12];
  assert.equal(row.slice(5, 35), 'X'.repeat(30));
});

test('rect fills every cell between the corners, either way round', () => {
  assert.equal(M.rect(3, 4, 1, 2).length, 9);
  const lv = M.blank(20);
  const p = M.newPatch('rect');
  for (const [x, y] of M.rect(2, 5, 6, 7)) M.placeElement(lv, p, M.elementById('brick'), x, y);
  const g = grid(lv);
  for (let y = 5; y <= 7; y++) assert.equal(g[y].slice(2, 7), 'BBBBB');
  assert.equal(g[4].trim(), '');
});

test('placing a question block gives it a content; erasing removes it', () => {
  const lv = M.blank(20);
  const p = M.newPatch('q');
  M.placeElement(lv, p, M.elementById('question'), 5, 9, { kind: 'star', count: 1, pool: null });
  assert.equal(lv.contents.get('5,9').kind, 'star');
  M.eraseCell(lv, p, 5, 9);
  assert.equal(M.at(lv, 5, 9), ' ');
  assert.equal(lv.contents.has('5,9'), false);
});

test('fillGround: whole level, two layers, turf left to the renderer', () => {
  const lv = M.blank(40);
  M.fillGround(lv, { layers: 2 });
  const g = grid(lv);
  assert.equal(g[13], 'X'.repeat(40));
  assert.equal(g[14], 'X'.repeat(40));
  assert.equal(g[12].trim(), '');
});

test('fillGround: keepGaps leaves existing gaps open', () => {
  const lv = M.blank(80);
  M.fillGround(lv, { layers: 2 });
  for (let x = 60; x <= 62; x++) for (const y of [13, 14]) lv.tiles[y * 80 + x] = ' ';
  M.fillGround(lv, { layers: 3, keepGaps: true });
  const g = grid(lv);
  assert.equal(g[14].slice(60, 63), '   ', 'the gap was filled');
  assert.equal(g[12][10], 'X', 'other columns did not get the third layer');
  assert.equal(g[12][61], ' ');
});

test('fillGround: only the selected columns', () => {
  const lv = M.blank(60);
  M.fillGround(lv, { layers: 2, x0: 10, x1: 40 });
  const g = grid(lv);
  assert.equal(g[13].slice(10, 41), 'X'.repeat(31));
  assert.equal(g[13][9], ' ');
  assert.equal(g[13][41], ' ');
});

test('fillGround is one undo step', () => {
  const lv = M.blank(50);
  const h = new M.History();
  h.push(M.fillGround(lv, { layers: 2 }));
  h.undo(lv);
  assert.ok(lv.tiles.every(c => c === ' '));
});

test('fillGround leaves pipes, stairs and blocks whole', () => {
  const lv = M.starter(60);
  M.placeStamp(lv, M.newPatch('p'), 'pipe', 3, 20, 5);
  M.placeStamp(lv, M.newPatch('s'), 'stairs-up', 2, 30, 5);
  M.fillGround(lv, { layers: 4 });
  const g = grid(lv);
  assert.equal(g[12].slice(20, 22), '{}', 'the pipe bottom was overwritten');
  assert.equal(g[11].slice(20, 22), '{}');
  assert.equal(g[12].slice(30, 32), 'HH', 'the stairs were overwritten');
  assert.equal(g[11][10], 'X', 'open columns did not get the new layers');
});

test('fillGround never buries the spawn or an enemy', () => {
  const lv = M.starter(60);
  M.placeElement(lv, M.newPatch('g'), M.elementById('goomba'), 25, 12);
  const h = new M.History();
  h.push(M.fillGround(lv, { layers: 4 }));
  assert.equal(lv.spawn.y, 10, 'the spawn should stand on the new top at row 11');
  assert.equal(M.at(lv, lv.spawn.x, lv.spawn.y), ' ');
  assert.equal(M.at(lv, lv.spawn.x, lv.spawn.y + 1), 'X');
  const g = lv.entities.find(e => e.kind === 'goomba');
  assert.equal(g.y, 10);
  // And the lift is part of the same undo step.
  h.undo(lv);
  assert.equal(lv.spawn.y, 12);
  assert.equal(lv.entities.find(e => e.kind === 'goomba').y, 12);
});

// ------------------------------------------------------------ 7.3 stamps

test('pipe stamp: always whole, lip on top, standing on the ground', () => {
  for (const height of [1, 2, 3, 4, 7]) {
    const lv = M.blank(40);
    M.fillGround(lv, { layers: 2 });
    const p = M.newPatch('pipe');
    M.placeStamp(lv, p, 'pipe', height, 10, 5);
    const g = grid(lv);
    const top = 13 - height;
    assert.equal(g[top].slice(10, 12), '[]', `height ${height}: lip`);
    for (let y = top + 1; y < 13; y++) assert.equal(g[y].slice(10, 12), '{}', `height ${height}: body row ${y}`);
    assert.equal(g[top - 1].slice(10, 12), '  ', `height ${height}: nothing above`);
    // Every lip and body cell has its other half.
    for (let y = 0; y < 15; y++) {
      for (let x = 0; x < 40; x++) {
        const c = g[y][x];
        if (c === '[') assert.equal(g[y][x + 1], ']');
        if (c === '{') assert.equal(g[y][x + 1], '}');
      }
    }
  }
});

test('stairs stamp climbs from the ground', () => {
  const lv = M.blank(40);
  M.fillGround(lv, { layers: 2 });
  M.placeStamp(lv, M.newPatch('s'), 'stairs-up', 4, 20, 3);
  const g = grid(lv);
  assert.equal(g[12].slice(20, 24), 'HHHH');
  assert.equal(g[9].slice(20, 24), '   H');
});

test('gap stamp clears the ground the whole way down', () => {
  const lv = M.blank(40);
  M.fillGround(lv, { layers: 2 });
  M.placeStamp(lv, M.newPatch('g'), 'gap', 3, 15, 13);
  const g = grid(lv);
  assert.equal(g[13].slice(14, 19), 'X   X');
  assert.equal(g[14].slice(15, 18), '   ');
});

// ------------------------------------------------------------ 7.4 clipboard

test('copy and paste carry block contents and enemies', () => {
  const lv = M.blank(60);
  const p = M.newPatch('setup');
  M.placeElement(lv, p, M.elementById('brick'), 10, 9);
  M.placeElement(lv, p, M.elementById('question'), 11, 9, { kind: 'oneup', count: 1, pool: null });
  M.placeElement(lv, p, M.elementById('goomba'), 11, 12);
  const clip = M.copyRect(lv, { x0: 10, y0: 9, x1: 12, y1: 12 });
  M.paste(lv, M.newPatch('paste'), clip, 30, 9);
  assert.equal(M.at(lv, 30, 9), 'B');
  assert.equal(M.at(lv, 31, 9), '?');
  assert.equal(lv.contents.get('31,9').kind, 'oneup');
  assert.ok(lv.entities.some(e => e.kind === 'goomba' && e.x === 31 && e.y === 12));
  // The original is still there.
  assert.equal(lv.contents.get('11,9').kind, 'oneup');
});

test('moving a selection leaves its old place empty', () => {
  const lv = M.blank(40);
  const p = M.newPatch('m');
  M.placeElement(lv, p, M.elementById('question'), 5, 5, { kind: 'star', count: 1, pool: null });
  M.moveRect(lv, p, { x0: 5, y0: 5, x1: 5, y1: 5 }, 10, 2);
  assert.equal(M.at(lv, 5, 5), ' ');
  assert.equal(lv.contents.has('5,5'), false);
  assert.equal(M.at(lv, 15, 7), '?');
  assert.equal(lv.contents.get('15,7').kind, 'star');
});

// ------------------------------------------------------------ 7.5 properties

test('probabilities match the weights and add up to 100', () => {
  assert.deepEqual(M.probabilities([{ kind: 'mushroom', w: 3 }, { kind: 'oneup', w: 1 }]), [75, 25]);
  assert.deepEqual(M.probabilities([{ kind: 'a', w: 1 }, { kind: 'b', w: 1 }, { kind: 'c', w: 1 }]).reduce((a, b) => a + b), 100);
  assert.deepEqual(M.probabilities([{ kind: 'a', w: 0 }, { kind: 'b', w: 2 }]), [0, 100]);
  assert.deepEqual(M.probabilities([{ kind: 'a', w: 0 }]), [0]);
});

test('setting many blocks at once is one change and keeps hidden flags', () => {
  const lv = M.blank(40);
  const setup = M.newPatch('s');
  for (const x of [5, 6, 7, 8, 9]) M.placeElement(lv, setup, M.elementById('question'), x, 9);
  M.placeElement(lv, setup, M.elementById('hidden'), 12, 5);
  const h = new M.History();
  const p = M.newPatch('batch');
  const cells = M.blocksIn(lv, { x0: 0, y0: 0, x1: 20, y1: 14 });
  assert.equal(cells.length, 6);
  M.setContents(lv, p, cells, { kind: 'random', count: 1, pool: [{ kind: 'mushroom', w: 1 }, { kind: 'star', w: 1 }] });
  h.push(p);
  for (const [x, y] of cells) assert.equal(lv.contents.get(M.key(x, y)).kind, 'random');
  assert.equal(lv.contents.get('12,5').hidden, true);
  h.undo(lv);
  for (const x of [5, 6, 7, 8, 9]) assert.equal(lv.contents.get(`${x},9`).kind, 'coin');
});

test('there is only ever one spawn and one flag', () => {
  const lv = M.starter(80);
  const p = M.newPatch('f');
  const oldX = lv.flag.x;
  M.placeElement(lv, p, M.elementById('flag'), 40, 5);
  const g = grid(lv);
  const poles = g.reduce((n, row) => n + [...row].filter(c => c === 'F' || c === 'T').length, 0);
  assert.equal(poles, lv.flag.bottomY - lv.flag.topY + 1);
  assert.equal(g[lv.flag.topY][40], 'T');
  assert.equal(g[5][oldX], ' ');
  M.placeElement(lv, p, M.elementById('spawn'), 10, 12);
  M.placeElement(lv, p, M.elementById('spawn'), 20, 12);
  assert.deepEqual(lv.spawn, { x: 20, y: 12 });
});

test('resize: widen on the right keeps everything in place and extends the floor', () => {
  const lv = M.starter(100);
  const flagX = lv.flag.x;
  M.placeElement(lv, M.newPatch('b'), M.elementById('brick'), 30, 9);
  M.resize(lv, M.newPatch('w'), 150, 'right');
  assert.equal(lv.width, 150);
  assert.equal(M.at(lv, 30, 9), 'B');
  assert.equal(lv.flag.x, flagX);
  assert.equal(M.at(lv, 140, 13), 'X');
});

test('resize: shrinking reports the columns it would empty', () => {
  const lv = M.starter(100);
  M.placeElement(lv, M.newPatch('b'), M.elementById('brick'), 95, 9);
  const lost = M.lostByResize(lv, 90, 'right');
  assert.ok(lost.includes(95), 'the brick column is not reported');
  assert.ok(lost.includes(lv.flag.x), 'the flag column is not reported');
  assert.equal(M.lostByResize(lv, 120, 'right').length, 0);
});

test('resize: adding on the left shifts everything right', () => {
  const lv = M.starter(60);
  const p = M.newPatch('b');
  M.placeElement(lv, p, M.elementById('goomba'), 20, 12);
  const spawn = lv.spawn.x;
  M.resize(lv, M.newPatch('w'), 70, 'left');
  assert.ok(lv.entities.some(e => e.x === 30));
  assert.equal(lv.spawn.x, spawn + 10);
});

// ------------------------------------------------------------ 7.7 undo

test('a whole drag undoes in one step', () => {
  const lv = M.blank(60);
  const h = new M.History();
  const p = M.newPatch('pen');
  for (const [x, y] of M.line(0, 12, 40, 12)) M.placeElement(lv, p, M.elementById('ground'), x, y);
  h.push(p);
  assert.equal(grid(lv)[12].trim().length, 41);
  h.undo(lv);
  assert.equal(grid(lv)[12].trim(), '');
  h.redo(lv);
  assert.equal(grid(lv)[12].trim().length, 41);
});

test('undoing a content change restores the old content', () => {
  const lv = M.blank(20);
  const h = new M.History();
  const p0 = M.newPatch('q');
  M.placeElement(lv, p0, M.elementById('question'), 5, 9);
  h.push(p0);
  const p1 = M.newPatch('kind');
  M.setContent(lv, p1, 5, 9, { kind: 'star', count: 1, hidden: false, pool: null });
  h.push(p1);
  assert.equal(lv.contents.get('5,9').kind, 'star');
  h.undo(lv);
  assert.equal(lv.contents.get('5,9').kind, 'coin');
});

test('history keeps at least 200 steps', () => {
  const lv = M.blank(300);
  const h = new M.History();
  for (let i = 0; i < 250; i++) {
    const p = M.newPatch('step');
    M.setTile(lv, p, i, 12, 'X');
    h.push(p);
  }
  let n = 0;
  while (h.undo(lv)) n++;
  assert.ok(n >= 200, `only ${n} steps`);
});

test('an edit after undo clears redo', () => {
  const lv = M.blank(20);
  const h = new M.History();
  const a = M.newPatch('a'); M.setTile(lv, a, 1, 1, 'B'); h.push(a);
  h.undo(lv);
  const b = M.newPatch('b'); M.setTile(lv, b, 2, 2, 'B'); h.push(b);
  assert.equal(h.canRedo, false);
});

test('undoing a resize restores the old width and contents', () => {
  const lv = M.starter(100);
  const before = M.toFile(lv);
  const h = new M.History();
  const p = M.newPatch('w');
  M.resize(lv, p, 60, 'right');
  h.push(p);
  h.undo(lv);
  assert.equal(M.toFile(lv), before);
});

// ------------------------------------------------------------ file round trip

test('a level survives fromFile(toFile(x))', () => {
  const lv = M.starter(90, 'snow');
  const p = M.newPatch('x');
  M.placeElement(lv, p, M.elementById('question'), 20, 9, { kind: 'random', count: 1, pool: [{ kind: 'star', w: 2 }, { kind: 'coin', w: 1 }] });
  M.placeElement(lv, p, M.elementById('koopa'), 40, 12);
  lv.meta = { title: '雪原', author: 'tom', difficulty: 2 };
  const again = M.fromFile(M.toFile(lv));
  assert.equal(M.toFile(again), M.toFile(lv));
  assert.deepEqual(again.contents.get('20,9').pool, [{ kind: 'star', w: 2 }, { kind: 'coin', w: 1 }]);
});

test('ids: suggestions are valid, bad ids are refused', () => {
  assert.equal(M.suggestId('Tom', 'Sky Run!'), 'c-tom-sky-run');
  assert.ok(M.validId(M.suggestId('', '')));
  for (const bad of ['My Level!', 'c-', 'tom-level', '../x']) assert.equal(M.validId(bad), false, bad);
  for (const good of ['1-2', 'c-tom-sky-run']) assert.ok(M.validId(good), good);
});
