/* AdminX iframe/new-window client. No framework dependency; token stays in memory. */
(() => {
  'use strict';
  const config = JSON.parse(document.getElementById('adminx-config').textContent);
  const origins = config.platformOrigins;
  const host = window.parent !== window ? window.parent : window.opener;
  const listeners = new Map();
  const pending = new Map();
  let origin = null;
  let token = '';
  let tokenFlight = null;
  let connected = false;
  let epoch = 0;
  let sequence = 0;
  const clientID = Math.random().toString(36).slice(2);
  let controller = new AbortController();
  try { localStorage.removeItem('business_token'); } catch { /* remove legacy persisted credential */ }
  const emit = (name, value) => listeners.get(name)?.forEach(fn => fn(value));
  function rpc(method, payload = {}) {
    if (!host || host.closed) return Promise.reject(new Error('请从 AdminX 平台进入业务页面'));
    if (pending.size >= 30) return Promise.reject(new Error('平台请求过多，请稍后重试'));
    return new Promise((resolve, reject) => {
      const id = `business-${clientID}-${Date.now()}-${++sequence}`;
      const timer = setTimeout(() => { pending.delete(id); reject(new Error('平台连接超时，请确认已登录并重试')); }, method === 'confirm' ? 300000 : 30000);
      pending.set(id, { resolve, reject, timer });
      const packet = { type: 'adminx:request', version: 1, id, method, payload };
      try { (origin ? [origin] : origins).forEach(target => host.postMessage(packet, target)); }
      catch (error) { clearTimeout(timer); pending.delete(id); reject(error); }
    });
  }
  function endSession(reason) {
    epoch++;
    token = '';
    connected = false;
    tokenFlight = null;
    controller.abort();
    for (const item of pending.values()) { clearTimeout(item.timer); item.reject(new Error('平台会话已结束')); }
    pending.clear();
    emit('logout', reason);
  }
  function onMessage(event) {
    if (event.source !== host || !origins.includes(event.origin) || (origin && event.origin !== origin)) return;
    const data = event.data;
    if (data?.type === 'adminx:response') {
      const item = pending.get(data.id);
      if (!item) return;
      origin = event.origin;
      pending.delete(data.id);
      clearTimeout(item.timer);
      if (typeof data.error === 'string') item.reject(new Error(data.error));
      else item.resolve(data.result);
    } else if (data?.type === 'adminx:event') {
      if (data.event === 'logout' || data.event === 'disconnect') endSession(data.event);
      else if (connected && data.event === 'theme') emit('theme', data.payload);
    }
  }
  window.addEventListener('message', onMessage);
  async function start() {
    const session = epoch;
    const context = await rpc('context');
    if (session !== epoch) throw new Error('平台会话已变更');
    if (typeof context?.token !== 'string' || !context.token) throw new Error('平台未提供有效登录状态');
    controller = new AbortController();
    token = context.token;
    connected = true;
    emit('theme', context.theme);
    return context;
  }
  function renew(rejectedToken) {
    if (tokenFlight) return tokenFlight;
    const session = epoch;
    tokenFlight = rpc('token', { rejectedToken }).then(result => {
      if (session !== epoch || !connected || !result?.token) throw new Error('平台会话已结束');
      token = result.token;
      return token;
    }).finally(() => { if (session === epoch) tokenFlight = null; });
    return tokenFlight;
  }
  async function request(path, options = {}) {
    if (!connected || !token) throw new Error('请先连接 AdminX 平台');
    const base = new URL('.', location.href);
    const url = new URL(path, base);
    if (url.origin !== location.origin || !url.pathname.startsWith(base.pathname + 'api/v1/')) throw new Error('业务请求地址不受信任');
    const session = epoch;
    for (let attempt = 0; attempt < 2; attempt++) {
      const usedToken = token;
      const headers = new Headers(options.headers);
      headers.set('Authorization', `Bearer ${usedToken}`);
      if (options.body) headers.set('Content-Type', 'application/json');
      const response = await fetch(url, { ...options, headers, credentials: 'omit', signal: controller.signal, redirect: 'error' });
      let body;
      try { body = await response.json(); } catch { throw new Error('业务响应格式错误，请检查服务或网关'); }
      if (session !== epoch) throw new Error('平台会话已结束');
      if (response.status === 401 || body.code === 401) {
        if (attempt === 0) { await renew(usedToken); continue; }
        throw new Error('业务认证失败，请重新从平台进入');
      }
      if (!response.ok || (typeof body.code === 'number' && body.code >= 400)) {
        throw new Error(typeof body.msg === 'string' ? body.msg : '业务请求失败');
      }
      return body.data;
    }
  }
  let lastActivity = 0;
  const activity = event => {
    if (!event.isTrusted || !connected || Date.now() - lastActivity < 5000) return;
    lastActivity = Date.now();
    void rpc('activity').catch(() => {});
  };
  const events = ['pointerdown', 'pointermove', 'keydown', 'wheel', 'touchstart'];
  events.forEach(name => window.addEventListener(name, activity, { passive: true }));
  window.addEventListener('pagehide', () => {
    endSession('closed');
  });
  window.addEventListener('pageshow', event => { if (event.persisted) location.reload(); });
  window.AdminXBridge = Object.freeze({
    start, request,
    on(name, fn) {
      if (!listeners.has(name)) listeners.set(name, new Set());
      listeners.get(name).add(fn);
      return () => listeners.get(name)?.delete(fn);
    },
    ready: () => rpc('ready'),
    route: hash => rpc('route', { hash }),
    dirty: value => rpc('dirty', { value }),
    navigate: code => rpc('navigate', { code }),
    confirm: text => rpc('confirm', { text }),
    notify: text => rpc('message', { text }),
    platformURL: config.platformURL,
  });
})();
