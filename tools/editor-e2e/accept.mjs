// Acceptance 9.1, browser half: build a level the way a contributor would and
// save it. The Go test (accept_test.go in the worktree) then plays it.
import { launch, sleep } from './cdp.mjs';

const b = await launch(process.argv[2]);
let fails = 0;
const ok = (c, m) => { console.log((c ? 'PASS ' : 'FAIL ') + m); if (!c) fails++; };
const at = async (x, y) => { await b.evaluate(`__editor.reveal(${x})`); return b.evaluate(`__editor.cellCentre(${x}, ${y})`); };
const click = async (x, y) => {
  const [px, py] = await at(x, y);
  await b.mouse('mouseMoved', px, py, 'none', 0);
  await b.mouse('mousePressed', px, py); await b.mouse('mouseReleased', px, py, 'left', 0);
  await sleep(60);
};
const toast = () => b.evaluate(`document.getElementById('toast').textContent`);

try {
  for (let i = 0; i < 80; i++) { await sleep(250); if (await b.evaluate(`!document.getElementById('loading')`)) break; }
  await b.evaluate(`localStorage.clear()`);

  // New level: snow theme, 80 wide.
  await b.evaluate(`document.getElementById('btn-new').click()`); await sleep(200);
  await b.evaluate(`document.querySelector('#new-themes [data-theme=snow]').click()`);
  await b.evaluate(`document.getElementById('new-width').value = '80'`);
  await b.evaluate(`document.getElementById('btn-new-ok').click()`); await sleep(400);

  // Gradient sky through the custom pickers.
  await b.evaluate(`(() => {
    const top = document.getElementById('sky-top'), bottom = document.getElementById('sky-bottom');
    top.value = '#1b2a4a'; bottom.value = '#f3a65a';
    const r = document.querySelector('input[name=sky-mode][value=grad]'); r.checked = true; r.dispatchEvent(new Event('change'));
  })()`);
  await sleep(200);

  // Fill ground, one layer deeper than the starter.
  await b.evaluate(`document.getElementById('fill-layers').value = '3'`);
  await b.key('f', 'KeyF', 0, 'f'); await sleep(150);

  // A pipe, height 2.
  await b.key('s', 'KeyS', 0, 's');
  await b.evaluate(`[...document.querySelectorAll('.stamp')].find(b => b.textContent === '水管').click()`);
  await click(30, 5);

  // A fixed block (1UP) and a random block (mushroom / star).
  await b.key('3', 'Digit3', 0, '3');
  await click(15, 7);
  await sleep(150);
  await b.evaluate(`[...document.querySelectorAll('#kind-area .kind-pick button')].find(b => b.textContent.includes('1UP')).click()`);
  await sleep(150);
  await b.key('3', 'Digit3', 0, '3');
  await click(18, 7);
  await sleep(150);
  await b.evaluate(`document.getElementById('mode-random').click()`); await sleep(150);
  await b.evaluate(`(async () => {
    for (const name of ['金币袋', '毒蘑菇', '1UP']) {
      const r = [...document.querySelectorAll('#props .kind')].find(r => r.textContent.includes(name));
      const c = r.querySelector('input[type=checkbox]'); c.checked = false; c.dispatchEvent(new Event('change'));
      await new Promise(r => setTimeout(r, 60));
    }
    const r = [...document.querySelectorAll('#props .kind')].find(r => r.textContent.trim().startsWith('金币') && !r.textContent.includes('袋'));
    const c = r.querySelector('input[type=checkbox]'); c.checked = false; c.dispatchEvent(new Event('change'));
  })()`);
  await sleep(200);

  // An enemy, well clear of the start.
  await b.key('6', 'Digit6', 0, '6');
  await click(45, 11);

  const lv = JSON.parse(await b.evaluate(`__editor.level()`));
  ok(lv.style.theme === 'snow' && lv.style.sky?.top === '#1b2a4a', 'snow theme with a gradient sky');
  ok(lv.contents.some(c => c.kind === 'oneup') && lv.contents.some(c => c.kind === 'random' && c.pool.length === 2), 'a fixed and a random block');
  ok(lv.entities.some(e => e.kind === 'goomba'), 'an enemy');
  ok(lv.rows[10].slice(30, 32) === '[]', 'a pipe');

  // Quick in-editor playtest: invincible, runs to the end.
  const end = await b.evaluate(`(() => {
    mario.start(__editor.level(), JSON.stringify({fromColumn: -1, invincible: true, zoom: 4, viewW: 30, viewH: 15, seed: 3}));
    let e = ''; for (let i = 0; i < 4000 && !e; i++) e = mario.step(2 | 16 | ((i % 40) < 18 ? 8 : 0), 1);
    return JSON.parse(mario.stop()).end;
  })()`);
  ok(end === 'cleared', 'playtest clears it (' + end + ')');

  // Meta and save.
  for (const [id, v] of [['level-title', '雪原小径'], ['level-author', 'tester'], ['level-id', 'c-tester-snow-path']]) {
    await b.evaluate(`(() => { const i = document.getElementById('${id}'); i.value = ${JSON.stringify(v)}; i.dispatchEvent(new Event('change')); })()`);
  }
  await b.evaluate(`(() => { const s = document.getElementById('level-diff'); s.value = '2'; s.dispatchEvent(new Event('change')); })()`);
  await sleep(300);
  const errs = await b.evaluate(`[...document.querySelectorAll('#issues li.error')].length`);
  ok(errs === 0, 'no errors before saving');
  await b.key('s', 'KeyS', 2, 's'); await sleep(800);
  ok((await toast()).startsWith('已保存'), 'saved: ' + await toast());
  await b.shot('shots/accept.png');
} finally {
  console.log('JS ERRORS:', b.errors.length ? b.errors : 'none');
  console.log(fails ? `${fails} FAILED` : 'ALL PASSED');
  b.close();
  process.exitCode = fails ? 1 : 0;
}
