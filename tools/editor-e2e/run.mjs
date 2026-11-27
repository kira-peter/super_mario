import { execFileSync, spawn } from 'node:child_process';
import { once } from 'node:events';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const tools = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(tools, '..', '..');
const binary = path.join(root, 'build', 'super-mario.exe');
const scratch = fs.mkdtempSync(path.join(os.tmpdir(), 'super-mario-e2e-'));

async function withEditor(name, test) {
  const dir = path.join(scratch, name);
  const builtin = path.join(dir, 'builtin');
  const local = path.join(dir, 'local');
  fs.mkdirSync(builtin, { recursive: true });
  fs.copyFileSync(path.join(root, 'pkg', 'level', 'data', '1-1.json'), path.join(builtin, '1-1.json'));
  const editor = spawn(binary, ['edit', '--no-browser', '--dir', builtin, '--local', local,
    '--draft-dir', path.join(dir, 'drafts')], { windowsHide: true, stdio: ['ignore', 'pipe', 'inherit'] });
  const finished = once(editor, 'close');
  try {
    const url = await new Promise((resolve, reject) => {
      let output = '';
      const timer = setTimeout(() => reject(new Error('Editor did not start: ' + output)), 10000);
      editor.stdout.on('data', chunk => {
        output += chunk;
        const address = output.match(/http:\/\/127\.0\.0\.1:\d+\//);
        if (address) { clearTimeout(timer); resolve(address[0]); }
      });
      finished.then(([code]) => {
        clearTimeout(timer);
        reject(new Error('Editor stopped with code ' + code + ': ' + output));
      }, reject);
    });
    await test(url, { ...process.env, BUILTIN_DIR: builtin, LOCAL_DIR: local });
  } finally {
    editor.kill();
    await finished;
  }
}

try {
  const version = fs.readFileSync(path.join(root, 'VERSION'), 'utf8').trim();
  const commit = execFileSync('git', ['rev-parse', '--short', 'HEAD'], { cwd: root, encoding: 'utf8' }).trim();
  const reported = execFileSync(binary, ['version'], { encoding: 'utf8' }).trim();
  if (reported !== `v${version} (${commit})`) throw new Error('Unexpected binary version: ' + reported);
  await withEditor('smoke', async url => {
    const info = await (await fetch(url + 'api/info')).json();
    if (!info.wasm || info.version !== reported) throw new Error('Embedded editor assets or version missing');
    const response = await fetch(url + 'editor.wasm');
    const bytes = await response.arrayBuffer();
    if (!response.ok || !WebAssembly.validate(bytes)) throw new Error('Invalid embedded editor.wasm');
  });
  if (!process.argv.includes('--smoke')) {
    fs.mkdirSync(path.join(tools, 'shots'), { recursive: true });
    for (const script of ['e2e.mjs', 'e2e2.mjs', 'e2e3.mjs', 'e2e-save.mjs']) {
      await withEditor(script, (url, env) => execFileSync(process.execPath, [script, url], {
        cwd: tools, env, stdio: 'inherit', timeout: 120000,
      }));
    }
  }
} finally {
  fs.rmSync(scratch, { recursive: true, force: true });
}
