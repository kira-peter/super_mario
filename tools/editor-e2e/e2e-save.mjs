import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { launch, sleep } from './cdp.mjs';

const url = process.argv[2];
const builtin = process.env.BUILTIN_DIR;
const originalPath = path.join(builtin, '1-1.json');
const tallPath = path.join(builtin, 'c-test-tall.json');
const tall = JSON.parse(fs.readFileSync(originalPath, 'utf8'));
tall.rows.push('X'.repeat(tall.width));
tall.height++;
const tallText = JSON.stringify(tall);
fs.writeFileSync(tallPath, tallText);

const b = await launch(url + '#1-1');
async function waitFor(expression) {
  for (let i = 0; i < 100; i++) {
    if (await b.evaluate(expression)) return;
    await sleep(100);
  }
  throw new Error('Timed out waiting for ' + expression);
}

function author(value) {
  return b.evaluate(`(() => {
    const input = document.getElementById('level-author');
    input.value = ${JSON.stringify(value)};
    input.dispatchEvent(new Event('change'));
  })()`);
}

try {
  await waitFor("!document.getElementById('loading')");
  await b.evaluate(`(() => {
    const fetchFile = window.fetch;
    window.fetch = async (url, options) => {
      const response = await fetchFile(url, options);
      if (options?.method === 'PUT' && url.startsWith('api/levels/')) {
        await new Promise(resolve => { window.releaseSave = resolve; });
      }
      return response;
    };
  })()`);
  await author('first version');
  await b.evaluate("document.getElementById('btn-save').click()");
  await waitFor('!!window.releaseSave');
  await author('newer edit');
  await b.evaluate('window.releaseSave()');
  await waitFor("!document.getElementById('btn-save').disabled");
  assert.equal(await b.evaluate("document.getElementById('dirty').hidden"), false);
  assert.equal(JSON.parse(fs.readFileSync(originalPath, 'utf8')).meta.author, 'first version');
  await waitFor(`(async () => {
    const response = await fetch('api/drafts?key=1-1');
    return response.status === 200 && JSON.parse((await response.json()).level).meta.author === 'newer edit';
  })()`);
  console.log('PASS a delayed save keeps newer edits dirty and recoverable');

  await b.evaluate("window.releaseSave = null; document.getElementById('btn-save').click()");
  await waitFor('!!window.releaseSave');
  await b.evaluate('window.releaseSave()');
  await waitFor("document.getElementById('dirty').hidden");
  assert.equal(JSON.parse(fs.readFileSync(originalPath, 'utf8')).meta.author, 'newer edit');

  await b.evaluate("document.getElementById('btn-open').click()");
  const tallButton = `document.querySelector('#open-list button[data-key="c-test-tall"]')`;
  await waitFor(`!!${tallButton}`);
  await b.evaluate(`${tallButton}.click()`);
  await waitFor("document.getElementById('toast').textContent.includes('15')");
  assert.equal(await b.evaluate('__editor.where().openedAs'), '1-1');
  assert.equal(fs.readFileSync(tallPath, 'utf8'), tallText);
  console.log('PASS opening a taller level leaves the current document and the taller file intact');
} finally {
  b.close();
}
