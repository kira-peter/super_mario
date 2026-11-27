import { launch, sleep } from './cdp.mjs';
import fs from 'node:fs';

const url = process.argv[2];
const b = await launch(url);
let fails = 0;
const ok = (cond, msg) => { console.log((cond ? 'PASS ' : 'FAIL ') + msg); if (!cond) fails++; };
const level = async () => JSON.parse(await b.evaluate(`__editor.level()`));
const row = (lv, y) => lv.rows[y];
const at = async (x, y) => { await b.evaluate(`__editor.reveal(${x})`); return b.evaluate(`__editor.cellCentre(${x}, ${y})`); };
const click = async (x, y, button = 'left') => {
  const [px, py] = await at(x, y);
  await b.mouse('mouseMoved', px, py, 'none', 0);
  await b.mouse('mousePressed', px, py, button, button === 'right' ? 2 : 1);
  await b.mouse('mouseReleased', px, py, button, 0);
  await sleep(60);
};
const drag = async (x0, y0, x1, y1) => {
  // Bring the middle of the drag into view first, then measure both ends in
  // the same view.
  await b.evaluate(`__editor.reveal(${Math.round((x0 + x1) / 2)})`);
  const [ax, ay] = await b.evaluate(`__editor.cellCentre(${x0}, ${y0})`);
  const [bx, by] = await b.evaluate(`__editor.cellCentre(${x1}, ${y1})`);
  await b.mouse('mouseMoved', ax, ay, 'none', 0);
  await b.mouse('mousePressed', ax, ay);
  await b.mouse('mouseMoved', bx, by);
  await b.mouse('mouseReleased', bx, by, 'left', 0);
  await sleep(80);
};
const toast = () => b.evaluate(`document.getElementById('toast').textContent`);
const setInput = (id, v) => b.evaluate(`(() => { const i = document.getElementById('${id}'); i.value = ${JSON.stringify(v)}; i.dispatchEvent(new Event('change')); })()`);

try {
  for (let i = 0; i < 80; i++) { await sleep(250); if (await b.evaluate(`!document.getElementById('loading')`)) break; }
  ok(await b.evaluate(`!document.getElementById('loading')`), 'engine loaded');
  // This suite checks messages in Chinese.
  await b.evaluate(`(() => { const s = document.getElementById('lang'); s.value = 'zh'; s.dispatchEvent(new Event('change')); })()`);
  const v = await b.evaluate(`__editor.view()`);
  ok(v.scale >= 24 && v.scale <= 56, `default zoom is comfortable (scale ${v.scale})`);
  await b.shot('shots/01-start.png');

  await sleep(400);
  ok((await b.evaluate(`document.getElementById('issue-count').textContent`)) === '', 'starter level has no issues');

  // Layout: Choose leads the tools and is the default; the property panel sits
  // in the left column, where the stamps used to be.
  const order = await b.evaluate(`[...document.querySelectorAll('#tools .tool')].map(x => x.dataset.tool)`);
  ok(order[0] === 'hand', 'Choose is the first tool: ' + order.join(','));
  ok((await b.evaluate(`document.querySelector('#tools .tool.on')?.dataset.tool`)) === 'hand', 'Choose is the default tool');
  ok((await b.evaluate(`[...document.querySelectorAll('aside.left section h3')].map(x => x.textContent.trim()).join('/')`)) === '工具/元素/属性/铺地面',
    'the left column is tools, elements, properties, ground');
  ok(await b.evaluate(`!document.getElementById('stamps')`), 'stamps are no longer a separate section');

  // Pen, fast drag: bricks across 8..20 on row 8 in one mouse move.
  await b.key('2', 'Digit2', 0, '2');
  await drag(8, 8, 20, 8);
  ok(row(await level(), 8).slice(8, 21) === 'B'.repeat(13), 'fast drag left no gaps (13 bricks)');

  await click(14, 8, 'right');
  ok(row(await level(), 8)[14] === ' ', 'right click erased a brick');

  // Pipe stamp, height 3, always whole.
  await b.key('s', 'KeyS', 0, 's');
  await b.key(']', 'BracketRight', 0, ']');
  await click(30, 5);
  let lv = await level();
  ok(row(lv, 10).slice(30, 32) === '[]' && row(lv, 11).slice(30, 32) === '{}' && row(lv, 12).slice(30, 32) === '{}',
    'pipe stamp: 3 high, whole, on the ground');

  // Question block, then random with two candidates.
  await b.key('3', 'Digit3', 0, '3');
  await click(12, 4);
  await sleep(200);
  await b.evaluate(`document.getElementById('mode-random').click()`); await sleep(150);
  await b.evaluate(`(async () => {
    const rows = () => [...document.querySelectorAll('#props .kind')];
    for (const name of ['金币袋', '毒蘑菇', '星星']) {
      const r = rows().find(r => r.textContent.includes(name)); const c = r.querySelector('input[type=checkbox]');
      c.checked = false; c.dispatchEvent(new Event('change')); await new Promise(r => setTimeout(r, 60));
    }
    const coin = rows().find(r => r.textContent.trim().startsWith('金币') && !r.textContent.includes('袋'));
    const cc = coin.querySelector('input[type=checkbox]'); cc.checked = false; cc.dispatchEvent(new Event('change'));
    await new Promise(r => setTimeout(r, 60));
    const m = rows().find(r => r.textContent.includes('红蘑菇')).querySelector('input[type=number]');
    m.value = '3'; m.dispatchEvent(new Event('change'));
  })()`);
  await sleep(200);
  const pcts = await b.evaluate(`[...document.querySelectorAll('#props .kind')].filter(r => r.querySelector('input[type=checkbox]').checked).map(r => r.textContent.replace(/\\s+/g, ''))`);
  ok(JSON.stringify(pcts).includes('75%') && JSON.stringify(pcts).includes('25%'), 'random panel shows 75% / 25%: ' + JSON.stringify(pcts));
  lv = await level();
  const q = lv.contents.find(c => c.x === 12 && c.y === 4);
  ok(q && q.kind === 'random' && q.pool.length === 2, 'block saved as random with a 2-entry pool');

  // The random weights fit the narrow left column: nothing is pushed out of
  // view, and the column does not scroll sideways.
  const fits = await b.evaluate(`(() => {
    const aside = document.querySelector('aside.left'), box = aside.getBoundingClientRect();
    const out = [...aside.querySelectorAll('#props button, #props input, #props .pct')]
      .filter(el => { const r = el.getBoundingClientRect(); return r.width === 0 || r.right > box.right + 1 || r.left < box.left - 1; });
    return { out: out.length, hscroll: aside.scrollWidth > aside.clientWidth + 1 };
  })()`);
  ok(fits.out === 0 && !fits.hscroll, 'the weights panel fits the left column: ' + JSON.stringify(fits));

  for (let i = 0; i < 5; i++) { await b.key('z', 'KeyZ', 2, 'z'); await sleep(80); }
  await sleep(150);
  ok((await level()).contents.find(c => c.x === 12 && c.y === 4).pool.length === 6, 'undo walks back the pool edits');
  for (let i = 0; i < 5; i++) await b.key('Z', 'KeyZ', 2 | 8, 'Z');
  await sleep(150);
  ok((await level()).contents.find(c => c.x === 12 && c.y === 4).pool.length === 2, 'redo puts them back');

  // --- the property panel follows the tool, not just the selection.
  // The pen on a question block edits what the next one will hold.
  await b.key('Escape', 'Escape'); await sleep(100);
  await b.key('3', 'Digit3', 0, '3'); await sleep(200);
  ok((await b.evaluate(`document.getElementById('props').innerText`)).includes('红蘑菇'),
    'pen on a question block offers the contents before anything is placed');
  await b.evaluate(`[...document.querySelectorAll('#kind-area .kind-pick button')].find(x => x.textContent.includes('星星')).click()`);
  await sleep(200);
  await click(60, 4);
  const starBlock = (await level()).contents.find(c => c.x === 60 && c.y === 4);
  ok(starBlock && starBlock.kind === 'star', 'what the pen panel picked is what the next block holds: ' + JSON.stringify(starBlock));

  // Switching the pen to the hidden block moves the panel to the pen, even
  // though a block was selected a moment ago. What is drawn must be hidden,
  // and going back to the question element must not inherit that flag.
  await b.evaluate(`document.querySelector('#elements [data-element=hidden]').click()`); await sleep(300);
  ok((await b.evaluate(`document.getElementById('props').innerText`)).includes('隐藏块'),
    'switching the pen to a hidden block moves the panel to the pen');
  ok(await b.evaluate(`!document.getElementById('c-hidden')`),
    'the pen panel has no hidden checkbox; the element decides that');
  await b.evaluate(`[...document.querySelectorAll('#kind-area .kind-pick button')].find(x => x.textContent.includes('星星')).click()`);
  await sleep(200);
  await click(63, 4);
  const hiddenBlock = (await level()).contents.find(c => c.x === 63 && c.y === 4);
  ok(hiddenBlock && hiddenBlock.kind === 'star' && hiddenBlock.hidden === true,
    'the block drawn after the switch is a hidden star: ' + JSON.stringify(hiddenBlock));
  ok(row(await level(), 4)[63] === ' ', 'a hidden block leaves its tile as air');
  await b.evaluate(`document.querySelector('#elements [data-element=question]').click()`); await sleep(300);
  await b.evaluate(`[...document.querySelectorAll('#kind-area .kind-pick button')].find(x => x.textContent.includes('1UP')).click()`);
  await sleep(200);
  await click(66, 4);
  const visibleAgain = (await level()).contents.find(c => c.x === 66 && c.y === 4);
  ok(visibleAgain && visibleAgain.kind === 'oneup' && !visibleAgain.hidden && row(await level(), 4)[66] === '?',
    'going back to the question element draws a visible block: ' + JSON.stringify(visibleAgain));
  // A stamp shows its size in the panel, with no separate row in the sidebar.
  await b.key('s', 'KeyS', 0, 's'); await sleep(200);
  ok((await b.evaluate(`document.getElementById('props').innerText`)).includes('图章') &&
    await b.evaluate(`!!document.getElementById('stamp-size-n')`), 'the stamp panel carries the stamp size');
  ok(await b.evaluate(`!document.getElementById('stamp-size-row')`), 'the sidebar no longer keeps a stamp-size row');
  // Choose: click an existing block and edit it, without drawing.
  await b.key('h', 'KeyH', 0, 'h'); await sleep(150);
  await b.evaluate(`__editor.reveal(12)`); await sleep(100);
  const [rx, ry] = await at(12, 4);
  await b.mouse('mouseMoved', rx, ry, 'none', 0);
  await b.mouse('mousePressed', rx, ry); await b.mouse('mouseReleased', rx, ry, 'left', 0);
  await sleep(300);
  ok((await b.evaluate(`document.getElementById('props').innerText`)).includes('内容物'), 'Choose selects an existing block and shows its properties');
  ok(row(await level(), 4)[12] === '?', 'Choose did not paint over the block');

  // Gap stamp, then fill ground with 3 layers keeping gaps.
  await b.key('s', 'KeyS', 0, 's');
  await b.evaluate(`[...document.querySelectorAll('.stamp')].find(b => b.textContent === '坑').click()`);
  await click(50, 13);
  await b.evaluate(`document.getElementById('fill-layers').value = '3'`);
  await b.key('f', 'KeyF', 0, 'f'); await sleep(150);
  lv = await level();
  ok(row(lv, 12)[45] === 'X' && row(lv, 14)[50] === ' ', 'fill ground: third layer added, the gap kept');

  await b.evaluate(`document.querySelector('#themes [data-theme=snow]').click()`); await sleep(250);
  ok((await level()).style?.theme === 'snow', 'theme card applied');
  await b.shot('shots/02-edited.png');

  // Playtest; the level must be unchanged after.
  const before = await b.evaluate(`__editor.level()`);
  const [hx, hy] = await at(5, 5);
  await b.mouse('mouseMoved', hx, hy, 'none', 0);
  await b.key('Enter', 'Enter', 0, '\r'); await sleep(250);
  ok(await b.evaluate(`__editor.playing()`), 'Enter starts a playtest');
  // He arrives first (pixels assembling, then the drop, 88 steps) and cannot
  // move until he lands; wait for that before holding the keys.
  for (let i = 0; i < 100 && (await b.evaluate(`__editor.playSteps()`)) < 90; i++) await sleep(100);
  const arrived = await b.evaluate(`__editor.playSteps()`);
  await b.keyDown('ArrowRight', 'ArrowRight'); await b.keyDown('Shift', 'ShiftLeft');
  await sleep(600);
  const heldMid = await b.evaluate(`__editor.playKeys()`);
  await b.shot('shots/03-playing.png');
  const heldAfterShot = await b.evaluate(`__editor.playKeys()`);
  console.log(`  keys held mid-run ${heldMid}, after a screenshot ${heldAfterShot}`);
  await sleep(600);
  await b.key('z', 'KeyZ', 2, 'z');
  await b.keyUp('ArrowRight', 'ArrowRight'); await b.keyUp('Shift', 'ShiftLeft');
  ok(await b.evaluate(`__editor.playing()`), 'Ctrl+Z during play did not stop or undo');
  await b.key('Escape', 'Escape'); await sleep(250);
  ok(!(await b.evaluate(`__editor.playing()`)), 'Esc returns to editing');
  ok((await b.evaluate(`__editor.level()`)) === before, 'playtest left the level unchanged');
  const run = await b.evaluate(`__editor.lastRun()`);
  // Headless Chrome runs animation frames slower than real time, so judge the
  // movement by the steps the game actually ran after he arrived, not by the
  // wall clock.
  const moved = run ? run.steps - arrived : 0;
  ok(run && moved > 20 && run.column > 2 + moved * 0.02, `the held arrow key moved Mario (to column ${run && run.column.toFixed(1)} in ${moved} steps after arriving)`);
  const view = await b.evaluate(`__editor.view()`);
  const cols = await b.evaluate(`document.getElementById('canvas').width`) / view.scale;
  ok(Math.abs(view.x + cols / 2 - run.column) < 2 || view.x <= -1, 'the view is centred where the run ended');
  await b.shot('shots/04-trace.png');

  const [gx, gy] = await at(50, 6);
  await b.mouse('mouseMoved', gx, gy, 'none', 0);
  await b.key('Enter', 'Enter', 8, '\r'); await sleep(250);
  ok((await toast()).includes('没有落脚点'), 'play-from-here over a gap explains the fallback');
  await b.key('Escape', 'Escape'); await sleep(200);

  await setInput('level-id', 'My Level!');
  ok((await toast()).includes('小写字母'), 'bad id is refused with the rule');
  await setInput('level-id', 'c-test-snow-run'); await sleep(150);
  ok((await b.evaluate(`document.getElementById('st-where').textContent`)).includes('c-test-snow-run.json'), 'the status bar shows where it will be saved');
  await b.key('s', 'KeyS', 2, 's'); await sleep(700);
  ok((await toast()).startsWith('已保存'), 'Ctrl+S saved: ' + await toast());
  ok(await b.evaluate(`document.getElementById('dirty').hidden`), 'saved level is clean');

  const first = fs.readFileSync(process.env.BUILTIN_DIR + '/c-test-snow-run.json', 'utf8');
  await b.evaluate(`document.getElementById('btn-save').click()`); await sleep(600);
  ok(fs.readFileSync(process.env.BUILTIN_DIR + '/c-test-snow-run.json', 'utf8') === first, 'saving twice is byte-identical');

  await b.key('2', 'Digit2', 0, '2');
  await click(40, 3);
  await sleep(900);
  ok(await b.evaluate(`fetch('api/drafts?key=c-test-snow-run').then(r => r.status === 200)`), 'unsaved edit stored in app drafts');

  await b.key('?', 'Slash', 8, '?'); await sleep(150);
  ok(await b.evaluate(`document.getElementById('dlg-help').open`), '? opens the shortcut list');
  await b.key('Escape', 'Escape');
} finally {
  console.log('JS ERRORS:', b.errors.length ? b.errors : 'none');
  console.log(fails ? `${fails} FAILED` : 'ALL PASSED');
  b.close();
  process.exitCode = fails ? 1 : 0;
}
