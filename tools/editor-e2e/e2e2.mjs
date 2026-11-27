import { launch, sleep } from './cdp.mjs';

const b = await launch(process.argv[2]);
let fails = 0;
const ok = (cond, msg) => { console.log((cond ? 'PASS ' : 'FAIL ') + msg); if (!cond) fails++; };
const toast = () => b.evaluate(`document.getElementById('toast').textContent`);
const waitLoaded = async () => { for (let i = 0; i < 80; i++) { await sleep(250); if (await b.evaluate(`!document.getElementById('loading')`)) return; } };

try {
  await waitLoaded();
  await b.evaluate(`localStorage.clear()`);

  // A new level, unchanged, can be played to the flag: run right holding run
  // and jump in steps until the run ends, through the real WASM session.
  const cleared = await b.evaluate(`(() => {
    const lv = __editor.level();
    mario.start(lv, JSON.stringify({fromColumn: -1, zoom: 4, viewW: 30, viewH: 15, seed: 1}));
    let end = '';
    for (let i = 0; i < 3000 && !end; i++) end = mario.step(2 | 16, 1);
    const s = JSON.parse(mario.stop());
    return s.end;
  })()`);
  ok(cleared === 'cleared', 'a new level, untouched, runs straight to the flag (' + cleared + ')');

  // Every theme: the canvas sky and the playtest frame agree on the sky colour.
  const themes = await b.evaluate(`JSON.parse(mario.palettes()).themes.map(t => t.ID)`);
  ok(themes.length === 8, '8 themes offered');
  for (const id of themes) {
    await b.evaluate(`document.querySelector('#themes [data-theme=${id}]').click()`);
    await sleep(120);
    const same = await b.evaluate(`(() => {
      const lv = __editor.level();
      const sky = JSON.parse(mario.sky(lv, 4));
      mario.start(lv, JSON.stringify({fromColumn: -1, zoom: 4, viewW: 30, viewH: 15, seed: 1}));
      const f = mario.frame(); mario.stop();
      // Top-left pixel of the game frame is open sky at the frame's top row.
      const hex = '#' + [f.rgba[0], f.rgba[1], f.rgba[2]].map(v => v.toString(16).padStart(2, '0')).join('');
      const row = Math.round(f.cameraY * 4);
      return { hex, want: sky[row], row };
    })()`);
    ok(same.hex === same.want, `theme ${id}: playtest sky ${same.hex} matches the editor's ${same.want}`);
  }

  // An error blocks saving and playing, and says where.
  // Make one: a random block whose only candidate gets weight 0 cannot be
  // made in the panel (it refuses), so put a question block where the flag
  // pole stands... which the editor refuses too. Use the width instead: a
  // spawn outside the level is an error, made by editing the file directly
  // through the model is not possible from the UI either. So break the level
  // the one way the UI allows: a hidden block placed inside ground.
  const [gx, gy] = await b.evaluate(`(__editor.reveal(10), __editor.cellCentre(10, 13))`);
  await b.key('8', 'Digit8', 0, '8');
  await b.mouse('mouseMoved', gx, gy, 'none', 0);
  await b.mouse('mousePressed', gx, gy); await b.mouse('mouseReleased', gx, gy, 'left', 0);
  await sleep(300);
  const issues = await b.evaluate(`document.getElementById('issue-count').textContent`);
  console.log('  issues after a hidden block in the ground:', issues || 'none');

  // The UI keeps the level loadable whatever the user clicks; confirm by
  // checking there are no errors after a burst of random edits.
  const burst = await b.evaluate(`(async () => {
    const ids = ['ground','brick','question','stair','coin','goomba','koopa','hidden'];
    for (let i = 0; i < 200; i++) {
      const id = ids[i % ids.length];
      document.querySelector('[data-element=' + id + ']').click();
      const x = 3 + (i * 7) % 100, y = 2 + (i * 5) % 12;
      const [px, py] = (__editor.reveal(x), __editor.cellCentre(x, y));
      const c = document.getElementById('canvas');
      for (const t of ['pointerdown', 'pointerup']) c.dispatchEvent(new PointerEvent(t, { clientX: px, clientY: py, button: 0, pointerId: 1, bubbles: true }));
    }
    await new Promise(r => setTimeout(r, 400));
    return JSON.parse(mario.validate(__editor.level())).filter(i => i.severity === 'error').map(i => i.message);
  })()`);
  ok(burst.length === 0, '200 mixed clicks never produce a level the game refuses: ' + JSON.stringify(burst.slice(0, 3)));

  // Force an error the only way possible — through the file — and check the
  // save and play paths refuse it, naming the problem.
  await b.evaluate(`(() => {
    const lv = JSON.parse(__editor.level());
    lv.contents.push({ x: 0, y: 0, kind: 'random', pool: [{ kind: 'coin', w: 0 }] });
    lv.rows[0] = '?' + lv.rows[0].slice(1);
    localStorage.setItem('mario-editor-draft', JSON.stringify({ id: '', openedAs: '', level: JSON.stringify(lv), at: Date.now() }));
  })()`);
  // Two dialogs follow: "leave page?" (the level is dirty), then the
  // restore-draft confirm. Accept every dialog for a while; do not await the
  // reload itself, it waits on the first dialog.
  let accepted = 0;
  const t = setInterval(() => { b.send('Page.handleJavaScriptDialog', { accept: true }).then(() => accepted++, () => {}); }, 150);
  b.send('Page.reload').catch(() => {});
  for (let i = 0; i < 60 && accepted < 2; i++) await sleep(250);
  clearInterval(t);
  ok(accepted >= 2, `reload asked before leaving and offered to restore the draft (${accepted} dialogs)`);
  await waitLoaded();
  await sleep(400);
  const errText = await b.evaluate(`[...document.querySelectorAll('#issues li.error')].map(l => l.textContent).join(' | ')`);
  ok(errText.includes('add up to 0'), 'the restored draft shows its error: ' + errText.slice(0, 80));
  await b.key('s', 'KeyS', 2, 's'); await sleep(300);
  ok((await toast()).includes('修好之后才能保存'), 'saving is refused while there is an error');
  await b.key('Enter', 'Enter', 0, '\r'); await sleep(300);
  ok(!(await b.evaluate(`__editor.playing()`)) && (await toast()).includes('不能试玩'), 'playing is refused while there is an error');

  // Clicking the error jumps to it.
  await b.evaluate(`document.querySelector('#issues li.error').click()`); await sleep(200);
  const v = await b.evaluate(`__editor.view()`);
  ok(v.x <= 1, 'clicking the error scrolled to column 0');

  // Opening 1-1 and the trace and hidden toggle.
  await b.evaluate(`localStorage.clear()`);
} finally {
  console.log('JS ERRORS:', b.errors.length ? b.errors : 'none');
  console.log(fails ? `${fails} FAILED` : 'ALL PASSED');
  b.close();
  process.exitCode = fails ? 1 : 0;
}
