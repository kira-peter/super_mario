const LOCAL_KEY = 'mario-editor-draft';

function localDraft() {
  try {
    return JSON.parse(localStorage.getItem(LOCAL_KEY));
  } catch {
    return null;
  }
}

export async function readDraft(key) {
  let saved = null;
  try {
    const response = await fetch('api/drafts?' + new URLSearchParams({ key }));
    if (response.status === 200) saved = await response.json();
  } catch {
    // The same-origin recovery copy can still be available.
  }
  const local = localDraft();
  if (local?.level && (!key || local.openedAs === key) && (!saved || local.at >= saved.at)) return local;
  return saved;
}

let pending = Promise.resolve();

function request(token, options) {
  const result = pending.then(async () => {
    const response = await fetch('api/drafts/' + token, options);
    if (!response.ok) throw new Error('draft request failed');
  });
  pending = result.catch(() => {});
  return result;
}

export function storeDraft(draft) {
  try { localStorage.setItem(LOCAL_KEY, JSON.stringify(draft)); } catch { /* Use the app's copy. */ }
  const body = JSON.stringify(draft);
  return request(draft.token, {
    method: 'PUT', body, headers: { 'Content-Type': 'application/json' },
    keepalive: new TextEncoder().encode(body).byteLength < 60 * 1024,
  });
}

export function deleteDraft(token) {
  if (localDraft()?.token === token) {
    try { localStorage.removeItem(LOCAL_KEY); } catch { /* Use the app's copy. */ }
  }
  return request(token, { method: 'DELETE', keepalive: true });
}
