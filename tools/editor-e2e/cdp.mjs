// Minimal Chrome DevTools Protocol driver: no dependencies.
import { spawn } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

export async function launch(url, { width = 1500, height = 900 } = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'cdp-'));
  const executable = process.env.CHROME || {
    win32: 'C:/Program Files/Google/Chrome/Application/chrome.exe',
    darwin: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  }[process.platform] || 'google-chrome';
  const chrome = spawn(executable, [
    '--headless=new', '--remote-debugging-port=0', `--user-data-dir=${dir}`,
    `--window-size=${width},${height}`, '--no-first-run', '--no-default-browser-check', 'about:blank',
  ], { windowsHide: true, stdio: ['ignore', 'ignore', 'pipe'] });
  const wsUrl = await new Promise((res, rej) => {
    let buf = '';
    const timeout = setTimeout(() => rej(new Error('chrome did not start: ' + buf)), 15000);
    chrome.on('error', e => { clearTimeout(timeout); rej(e); });
    chrome.stderr.on('data', d => {
      buf += d;
      const m = buf.match(/ws:\/\/[^\s]+/);
      if (m) { clearTimeout(timeout); res(m[0]); }
    });
  });
  const port = new URL(wsUrl).port;
  const targets = await (await fetch(`http://127.0.0.1:${port}/json`)).json();
  const page = targets.find(t => t.type === 'page');
  const ws = new WebSocket(page.webSocketDebuggerUrl);
  await new Promise((resolve, reject) => {
    ws.addEventListener('open', resolve, { once: true });
    ws.addEventListener('error', e => reject(e.error || new Error('Chrome DevTools socket failed')), { once: true });
    ws.addEventListener('close', e => reject(new Error('Chrome DevTools socket closed: ' + e.code)), { once: true });
  });
  let id = 0; const pending = new Map(); const listeners = [];
  ws.addEventListener('message', ev => {
    const m = JSON.parse(ev.data);
    if (m.id && pending.has(m.id)) { const { res, rej } = pending.get(m.id); pending.delete(m.id); m.error ? rej(new Error(m.error.message)) : res(m.result); }
    else listeners.forEach(l => l(m));
  });
  const send = (method, params = {}) => new Promise((res, rej) => { const i = ++id; pending.set(i, { res, rej }); ws.send(JSON.stringify({ id: i, method, params })); });
  const errors = [];
  listeners.push(m => {
    if (m.method === 'Runtime.exceptionThrown') errors.push(m.params.exceptionDetails.exception?.description || m.params.exceptionDetails.text);
    if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') errors.push(m.params.args.map(a => a.value ?? a.description).join(' '));
    if (m.method === 'Log.entryAdded' && m.params.entry.level === 'error') errors.push(m.params.entry.text + ' ' + (m.params.entry.url || ''));
  });
  await send('Runtime.enable'); await send('Log.enable'); await send('Page.enable');
  await send('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile: false });
  await send('Page.navigate', { url });
  const evaluate = async (expr) => {
    const r = await send('Runtime.evaluate', { expression: expr, awaitPromise: true, returnByValue: true });
    if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description || r.exceptionDetails.text);
    return r.result.value;
  };
  const shot = async (file) => { const r = await send('Page.captureScreenshot', { format: 'png' }); fs.writeFileSync(file, Buffer.from(r.data, 'base64')); };
  const mouse = (type, x, y, button = 'left', buttons = 1, mods = 0) => send('Input.dispatchMouseEvent', { type, x, y, button, buttons, clickCount: 1, modifiers: mods });
  const key = async (key, code, mods = 0, text) => {
    await send('Input.dispatchKeyEvent', { type: 'keyDown', key, code, modifiers: mods, text, windowsVirtualKeyCode: vk(key, code) });
    await send('Input.dispatchKeyEvent', { type: 'keyUp', key, code, modifiers: mods, windowsVirtualKeyCode: vk(key, code) });
  };
  const keyDown = (key, code) => send('Input.dispatchKeyEvent', { type: 'keyDown', key, code, windowsVirtualKeyCode: vk(key, code) });
  const keyUp = (key, code) => send('Input.dispatchKeyEvent', { type: 'keyUp', key, code, windowsVirtualKeyCode: vk(key, code) });
  const close = () => { try { ws.close(); } catch {} chrome.kill(); };
  return { send, evaluate, shot, mouse, key, keyDown, keyUp, errors, close };
}

function vk(key, code) {
  const map = { Enter: 13, Escape: 27, ArrowRight: 39, ArrowLeft: 37, ArrowUp: 38, Space: 32, ShiftLeft: 16, Delete: 46, F5: 116 };
  if (map[code] || map[key]) return map[code] || map[key];
  if (key.length === 1) return key.toUpperCase().charCodeAt(0);
  return 0;
}

export const sleep = ms => new Promise(r => setTimeout(r, ms));
