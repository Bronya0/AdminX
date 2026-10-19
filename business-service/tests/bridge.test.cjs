const { test } = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');
const fs = require('node:fs');
const path = require('node:path');
const source = fs.readFileSync(path.join(__dirname, '../static/adminx-bridge.js'), 'utf8');

function client(methods = {}, fetcher = async () => ({ ok: true, status: 200, json: async () => ({ code: 200, data: 'ok' }) })) {
  const events = new Map();
  const packets = [];
  const origin = 'https://platform.example';
  const host = { closed: false, postMessage(packet, target) {
    packets.push({ packet, target });
    if (target !== origin) return;
    Promise.resolve().then(async () => {
      const handler = methods[packet.method];
      const result = handler ? await handler(packet.payload) : null;
      deliver({ type: 'adminx:response', id: packet.id, result });
    });
  }};
  function deliver(data, source = host, from = origin) {
    events.get('message')?.forEach(fn => fn({ source, origin: from, data }));
  }
  const window = { parent: host, opener: null, addEventListener(type, fn) {
    if (!events.has(type)) events.set(type, []);
    events.get(type).push(fn);
  }};
  const removed = [];
  const context = { window, document: { getElementById: () => ({ textContent: JSON.stringify({ platformOrigins: [origin], platformURL: origin }) }) },
    localStorage: { removeItem: key => removed.push(key) }, location: new URL('https://business.example/apps/demo/'),
    URL, Headers, AbortController, setTimeout, clearTimeout, Date, console, fetch: fetcher };
  vm.runInNewContext(source, context);
  return { sdk: window.AdminXBridge, deliver, packets, removed };
}

test('only trusted parent responses and events are accepted; no wildcard or persisted token', async () => {
  const c = client({ context: () => ({ token: 'access', theme: {} }) });
  let logouts = 0;
  c.sdk.on('logout', () => logouts++);
  await c.sdk.start();
  c.deliver({ type: 'adminx:event', event: 'logout' }, {}, 'https://platform.example');
  c.deliver({ type: 'adminx:event', event: 'logout' }, undefined, 'https://evil.example');
  assert.equal(logouts, 0);
  assert.equal(await c.sdk.request('api/v1/posts/'), 'ok');
  assert.deepEqual(c.removed, ['business_token']);
  assert.ok(c.packets.every(p => p.target === 'https://platform.example'));
  await assert.rejects(c.sdk.request('https://evil.example/api/v1/posts/'), /不受信任/);
  c.deliver({ type: 'adminx:event', event: 'logout' });
  assert.equal(logouts, 1);
  await assert.rejects(c.sdk.request('api/v1/posts/'), /连接/);
});

test('concurrent 401s share one renewal and retry once with the new access token', async () => {
  let renewals = 0;
  const attempts = [];
  const c = client({ context: () => ({ token: 'old' }), token: async payload => {
    assert.equal(payload.rejectedToken, 'old'); renewals++;
    await new Promise(resolve => setTimeout(resolve, 5));
    return { token: 'fresh' };
  }}, async (_url, options) => {
    const auth = options.headers.get('Authorization'); attempts.push(auth);
    return { ok: auth.endsWith('fresh'), status: auth.endsWith('fresh') ? 200 : 401,
      json: async () => ({ code: auth.endsWith('fresh') ? 200 : 401, data: 'ok' }) };
  });
  await c.sdk.start();
  assert.deepEqual(await Promise.all([c.sdk.request('api/v1/posts/'), c.sdk.request('api/v1/posts/')]), ['ok', 'ok']);
  assert.equal(renewals, 1);
  assert.equal(attempts.length, 4);
});

test('a second 401 fails without an infinite refresh loop', async () => {
  let renewals = 0;
  const c = client({ context: () => ({ token: 'old' }), token: () => { renewals++; return { token: 'fresh' }; } },
    async () => ({ ok: false, status: 401, json: async () => ({ code: 401 }) }));
  await c.sdk.start();
  await assert.rejects(c.sdk.request('api/v1/posts/'), /认证失败/);
  assert.equal(renewals, 1);
});

test('an in-flight renewal cannot restore a logged out session', async () => {
  let release;
  let requested;
  const started = new Promise(resolve => { requested = resolve; });
  const c = client({ context: () => ({ token: 'old' }), token: () => { requested(); return new Promise(resolve => { release = resolve; }); } },
    async () => ({ ok: false, status: 401, json: async () => ({ code: 401 }) }));
  await c.sdk.start();
  const pending = c.sdk.request('api/v1/posts/');
  const failed = assert.rejects(pending, /会话已结束/);
  await started;
  c.deliver({ type: 'adminx:event', event: 'logout' });
  release({ token: 'too-late' });
  await failed;
  await assert.rejects(c.sdk.request('api/v1/posts/'), /连接/);
});
