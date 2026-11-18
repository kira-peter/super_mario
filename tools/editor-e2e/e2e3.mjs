// Folders and language: the editor lists both level folders, saves a level
// back where it came from, never downloads, and switches language without
// touching the level.
import { launch, sleep } from './cdp.mjs';
import fs from 'node:fs';
import path from 'node:path';

const b = await launch(process.argv[2]);
const BUILTIN = process.env.BUILTIN_DIR, LOCAL = process.env.LOCAL_DIR;
let fails = 0;
const ok = (c, m) => { console.log((c ? 'PASS ' : 'FAIL ') + m); if (!c) fails++; };
const toast = () => b.evaluate(`document.getElementById('toast').textContent`);
const lang = l => b.evaluate(`(() => { const s = document.getElementById('lang'); s.value = '${l}'; s.dispatchEvent(new Event('change')); })()`);
const text = id => b.evaluate(`document.getElementById('${id}').textContent`);

try {
  for (let i = 0; i < 80; i++) { await sleep(250); if (await b.evaluate(`!document.getElementById('loading')`)) break; }
  await b.evaluate(`localStorage.clear()`);
  await lang('zh');

  // A level only in the local folder, and a local 1-1 beside the built-in one.
  const one = fs.readFileSync(path.join(BUILTIN, '1-1.json'), 'utf8');
  fs.mkdirSync(LOCAL, { recursive: true });
  fs.writeFileSync(path.join(LOCAL, 'c-amy-hills.json'), one.replace('"timeLimit": 400,', '"timeLimit": 400,\n  "meta": {"title": "Hills"},'));
  fs.writeFileSync(path.join(LOCAL, '1-1.json'), one.replace('"timeLimit": 400,', '"timeLimit": 300,'));

  // --- open: both folders, grouped, the same name twice
  await b.evaluate(`document.getElementById('btn-open').click()`); await sleep(400);
  const keys = await b.evaluate(`[...document.querySelectorAll('#open-list button[data-key]')].map(b => b.dataset.key)`);
  ok(keys.includes('1-1') && keys.includes('local:1-1') && keys.includes('local:c-amy-hills'), 'open lists both folders, 1-1 twice: ' + keys.join(', '));
  const groups = await b.evaluate(`[...document.querySelectorAll('#open-list .open-group')].map(g => g.textContent)`);
  ok(groups.length === 2 && groups.some(g => g.includes(BUILTIN)) && groups.some(g => g.includes(LOCAL)), 'each folder is labelled with its path');

  // --- open the local 1-1, change it, save: it goes back to the local folder
  await b.evaluate(`document.querySelector('#open-list button[data-key="local:1-1"]').click()`); await sleep(500);
  ok((await b.evaluate(`__editor.where()`)).target === 'local', 'a local level saves to the local folder');
  ok((await text('level-where')).includes('本地'), 'the level panel says it is a local level');
  await b.evaluate(`(() => { const i = document.getElementById('level-author'); i.value = 'me'; i.dispatchEvent(new Event('change')); })()`);
  await b.key('s', 'KeyS', 2, 's'); await sleep(700);
  const t1 = await toast();
  ok(t1.includes(path.join(LOCAL, '1-1.json')) || t1.includes(LOCAL), 'toast gives the full local path: ' + t1);
  ok(fs.readFileSync(path.join(LOCAL, '1-1.json'), 'utf8').includes('"author": "me"'), 'the local 1-1 was changed');
  ok(!fs.readFileSync(path.join(BUILTIN, '1-1.json'), 'utf8').includes('"author": "me"'), 'the built-in 1-1 was not touched');

  // --- a new level saves to the built-in folder (running in the repository)
  await b.evaluate(`document.getElementById('btn-new').click()`); await sleep(200);
  ok((await text('new-where')).includes(BUILTIN), 'the new-level dialog says where it will be saved');
  await b.evaluate(`document.getElementById('btn-new-ok').click()`); await sleep(400);
  await b.evaluate(`(() => { const i = document.getElementById('level-id'); i.value = 'c-me-first'; i.dispatchEvent(new Event('change')); })()`);
  await b.key('s', 'KeyS', 2, 's'); await sleep(700);
  ok(fs.existsSync(path.join(BUILTIN, 'c-me-first.json')), 'a new level was written to the built-in folder');
  ok(!fs.existsSync(path.join(LOCAL, 'c-me-first.json')), 'and not to the local one');

  // --- language: English, everything follows, the level is unchanged
  const before = await b.evaluate(`__editor.level()`);
  await lang('en');
  await sleep(200);
  ok((await text('btn-save')) === 'Save' && (await text('btn-open')) === 'Open', 'buttons are in English');
  const tools = await b.evaluate(`[...document.querySelectorAll('.tool')].map(b => b.textContent).join(' ')`);
  ok(tools.includes('Pen') && tools.includes('Eraser'), 'tool names are in English: ' + tools);
  const themes = await b.evaluate(`[...document.querySelectorAll('#themes .theme-card span')].map(s => s.textContent).join(',')`);
  ok(themes.includes('Grassland') && themes.includes('Lava'), 'theme names are in English');
  ok((await b.evaluate(`__editor.level()`)) === before, 'switching language did not change the level');
  const zhLeft = await b.evaluate(`(() => {
    const walk = n => [...n.childNodes].flatMap(c => c.nodeType === 3 ? [c.textContent] : (c.nodeName === 'SCRIPT' || c.nodeName === 'OPTION' ? [] : walk(c)));
    return walk(document.body).filter(s => /[\\u4e00-\\u9fff]/.test(s)).slice(0, 5);
  })()`);
  ok(zhLeft.length === 0, 'no Chinese text left on the page: ' + JSON.stringify(zhLeft));

  // A check result in English.
  await b.evaluate(`(() => {
    const els = document.querySelectorAll('[data-element=goomba]'); els[0].click();
  })()`);
  const [ex, ey] = await b.evaluate(`(__editor.reveal(5), __editor.cellCentre(5, 12))`);
  await b.mouse('mouseMoved', ex, ey, 'none', 0);
  await b.mouse('mousePressed', ex, ey); await b.mouse('mouseReleased', ex, ey, 'left', 0);
  await sleep(500);
  const issue = await b.evaluate(`[...document.querySelectorAll('#issues li .msg')].map(m => m.textContent).join(' | ')`);
  ok(/enemy is within \d+ tiles of the start/.test(issue), 'a check result reads in English: ' + issue);

  // Remembered after a reload.
  await b.evaluate(`localStorage.removeItem('mario-editor-draft')`);
  let accepted = 0;
  const tm = setInterval(() => { b.send('Page.handleJavaScriptDialog', { accept: true }).then(() => accepted++, () => {}); }, 150);
  b.send('Page.reload').catch(() => {});
  for (let i = 0; i < 80; i++) { await sleep(250); try { if (await b.evaluate(`!!document.getElementById('lang') && !document.getElementById('loading')`)) break; } catch {} }
  clearInterval(tm);
  ok((await text('btn-save')) === 'Save', 'English is remembered after a reload');
  await lang('zh'); await sleep(150);
  ok((await text('btn-save')) === '保存', 'and back to Chinese');
} finally {
  console.log('JS ERRORS:', b.errors.length ? b.errors : 'none');
  console.log(fails ? `${fails} FAILED` : 'ALL PASSED');
  b.close();
  process.exitCode = fails ? 1 : 0;
}
