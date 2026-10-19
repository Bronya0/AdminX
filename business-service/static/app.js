'use strict';
const bridge = window.AdminXBridge;
const $ = id => document.getElementById(id);
const state = { page: 0, count: 0, posts: [], status: '', search: '', post: 0, dirty: false, online: false, busy: false };
const PAGE_SIZE = 5;
let loadSequence = 0;
let editor;
function element(tag, text, className) {
  const node = document.createElement(tag);
  if (text !== undefined) node.textContent = String(text);
  if (className) node.className = className;
  return node;
}
function button(text, action, className = 'btn outline') {
  const node = element('button', text, className);
  node.type = 'button';
  node.addEventListener('click', () => Promise.resolve(action()).catch(showError));
  node.disabled = state.busy;
  return node;
}
function showError(error) { $('errorMsg').textContent = error instanceof Error ? error.message : String(error); }
function setDirty(value) {
  if (state.dirty === value) return;
  state.dirty = value;
  void bridge.dirty(value).catch(() => {});
}
function syncRoute() {
  const params = new URLSearchParams();
  if (state.page) params.set('page', String(state.page));
  if (state.status) params.set('status', state.status);
  if (state.search) params.set('search', state.search);
  if (state.post) params.set('post', String(state.post));
  const hash = params.size ? '#' + params.toString() : '';
  history.replaceState(null, '', location.pathname + location.search + hash);
  void bridge.route(hash).catch(() => {});
}
function restoreRoute(hash) {
  const params = new URLSearchParams((hash || '').replace(/^#/, ''));
  state.page = Math.min(100000, Math.max(0, Number.parseInt(params.get('page'), 10) || 0));
  state.post = Math.max(0, Number.parseInt(params.get('post'), 10) || 0);
  state.status = ['draft', 'published'].includes(params.get('status')) ? params.get('status') : '';
  state.search = (params.get('search') || '').slice(0, 100);
  $('filterStatus').value = state.status;
  $('searchInput').value = state.search;
}
function applyTheme(theme) {
  if (!theme || typeof theme !== 'object') return;
  const root = document.documentElement;
  root.classList.toggle('dark', theme.isDark === true);
  root.style.colorScheme = theme.isDark ? 'dark' : 'light';
  if (/^#[0-9a-f]{6}$/i.test(theme.primaryColor)) {
    root.style.setProperty('--primary', theme.primaryColor);
    root.style.setProperty('--primary-bg', theme.primaryColor + '22');
  }
  if (Number.isFinite(theme.fontSize)) root.style.setProperty('--font-size', Math.min(24, Math.max(12, theme.fontSize)) + 'px');
  if (Number.isFinite(theme.borderRadius)) root.style.setProperty('--radius', Math.min(20, Math.max(0, theme.borderRadius)) + 'px');
}
bridge.on('theme', applyTheme);
bridge.on('logout', () => {
  state.online = false;
  state.posts = [];
  state.post = 0;
  state.dirty = false;
  editor?.remove(); editor = null;
  $('titleInput').value = ''; $('contentInput').value = '';
  $('postList').replaceChildren();
  $('businessPanel').hidden = true;
  $('connectionPanel').hidden = false;
  $('connectionText').textContent = '平台连接已结束，请登录平台后重新进入。';
  $('statusPill').textContent = '未连接';
  $('statsInfo').textContent = '';
});
async function loadPosts() {
  const seq = ++loadSequence;
  $('errorMsg').textContent = '';
  try {
    const params = new URLSearchParams({ skip: state.page * PAGE_SIZE, limit: PAGE_SIZE });
    if (state.status) params.set('status', state.status);
    if (state.search) params.set('search', state.search);
    const data = await bridge.request('api/v1/posts/?' + params);
    if (!state.online || seq !== loadSequence) return;
    state.posts = data.results;
    state.count = data.count;
    if (state.page > 0 && !state.posts.length) { state.page = Math.max(0, Math.ceil(state.count / PAGE_SIZE) - 1); return loadPosts(); }
    $('statsInfo').textContent = `共 ${state.count} 篇`;
    renderPosts(); syncRoute();
  } catch (error) { if (state.online && seq === loadSequence) showError(error); }
}
function renderPosts() {
  const container = $('postList');
  container.replaceChildren();
  if (!state.posts.length) container.append(element('div', '暂无文章', 'empty'));
  else {
    const wrapper = element('div', undefined, 'table-scroll');
    const table = element('table');
    const heading = element('tr');
    ['标题', '状态', '创建时间', '操作'].forEach(text => heading.append(element('th', text)));
    table.append(heading);
    state.posts.forEach(post => {
      const row = element('tr');
      const title = element('td'); title.append(element('strong', post.title));
      const excerpt = element('p', post.content.slice(0, 60)); title.append(excerpt);
      row.append(title);
      const status = element('td'); status.append(element('span', post.status === 'published' ? '已发布' : '草稿', 'status-badge ' + (post.status === 'published' ? 'published' : 'draft')));
      row.append(status, element('td', new Date(post.created_at).toLocaleString()));
      const actions = element('td');
      const group = element('div', undefined, 'actions');
      group.append(button('编辑', () => openEditor(post.id)), button('删除', () => deletePost(post.id), 'btn danger'));
      actions.append(group); row.append(actions); table.append(row);
    });
    wrapper.append(table); container.append(wrapper);
  }
  $('paginationBar').style.display = 'flex';
  $('pageInfo').textContent = `第 ${state.page + 1} / ${Math.max(1, Math.ceil(state.count / PAGE_SIZE))} 页`;
  $('prevBtn').disabled = state.page === 0 || state.busy;
  $('nextBtn').disabled = (state.page + 1) * PAGE_SIZE >= state.count || state.busy;
}
async function mayDiscard() { return !state.dirty || await bridge.confirm('还有未保存的内容，确定放弃？'); }
function closeEditor() {
  editor?.remove(); editor = null;
  state.post = 0; setDirty(Boolean($('titleInput').value.trim() || $('contentInput').value.trim())); syncRoute();
}
async function openEditor(id) {
  if (!await mayDiscard()) return;
  const post = await bridge.request('api/v1/posts/' + id);
  if (!state.online) return;
  $('titleInput').value = ''; $('contentInput').value = '';
  editor?.remove();
  editor = element('div', undefined, 'edit-form visible');
  editor.append(element('h2', '编辑文章 #' + id));
  const row = element('div', undefined, 'form-row');
  const title = element('input'); title.type = 'text'; title.value = post.title; title.maxLength = 100; title.setAttribute('aria-label', '文章标题');
  const content = element('input'); content.type = 'text'; content.value = post.content; content.maxLength = 500; content.setAttribute('aria-label', '文章内容');
  const status = element('select'); status.setAttribute('aria-label', '文章状态');
  status.append(new Option('草稿', 'draft'), new Option('已发布', 'published')); status.value = post.status;
  [title, content, status].forEach(input => input.addEventListener('input', () => { editor.dataset.dirty = 'true'; setDirty(true); }));
  row.append(title, content, status); editor.append(row);
  editor.append(button('保存', () => write(async () => {
    if (!title.value.trim()) throw new Error('标题不能为空');
    await bridge.request('api/v1/posts/' + id, { method: 'PUT', body: JSON.stringify({ title: title.value.trim(), content: content.value.trim(), status: status.value }) });
    closeEditor();
  }), 'btn'), button('取消', async () => { if (await mayDiscard()) closeEditor(); }));
  $('postList').before(editor); state.post = id; setDirty(false); syncRoute(); title.focus();
}
async function write(action) {
  if (state.busy) return;
  state.busy = true;
  document.querySelectorAll('#businessPanel button').forEach(node => { node.disabled = true; });
  try { await action(); $('successMsg').textContent = '操作成功'; await loadPosts(); }
  catch (error) { showError(error); }
  finally {
    state.busy = false;
    document.querySelectorAll('#businessPanel button').forEach(node => { node.disabled = false; });
    if (state.online) renderPosts();
  }
}
async function deletePost(id) {
  if (!await bridge.confirm('确定删除此文章？')) return;
  await write(async () => {
    await bridge.request('api/v1/posts/' + id, { method: 'DELETE' });
    if (state.post === id) closeEditor();
  });
}
$('createBtn').addEventListener('click', () => write(async () => {
  const title = $('titleInput').value.trim();
  if (!title) throw new Error('请输入标题');
  await bridge.request('api/v1/posts/', { method: 'POST', body: JSON.stringify({ title, content: $('contentInput').value.trim(), status: $('statusSelect').value }) });
  $('titleInput').value = ''; $('contentInput').value = ''; state.page = 0; setDirty(Boolean(editor?.dataset.dirty));
}));
['titleInput', 'contentInput', 'statusSelect'].forEach(id => $(id).addEventListener('input', () => setDirty(true)));
async function filter() {
  if (!await mayDiscard()) return;
  $('titleInput').value = ''; $('contentInput').value = ''; closeEditor();
  state.status = $('filterStatus').value; state.search = $('searchInput').value.trim(); state.page = 0;
  await loadPosts();
}
$('filterStatus').addEventListener('change', () => filter().catch(showError));
$('searchBtn').addEventListener('click', () => filter().catch(showError));
$('searchInput').addEventListener('keydown', event => { if (event.key === 'Enter') void filter().catch(showError); });
async function page(delta) { if (await mayDiscard()) { $('titleInput').value = ''; $('contentInput').value = ''; closeEditor(); state.page += delta; await loadPosts(); } }
$('prevBtn').addEventListener('click', () => page(-1).catch(showError));
$('nextBtn').addEventListener('click', () => page(1).catch(showError));
$('navigateBtn').addEventListener('click', () => { if ($('platformMenu').value) void bridge.navigate($('platformMenu').value).catch(showError); });
$('platformLink').href = bridge.platformURL;
async function connect() {
  $('connectBtn').disabled = true;
  $('connectionText').textContent = '正在连接 AdminX 平台…';
  try {
    const context = await bridge.start();
    state.online = true;
    $('platformMenu').replaceChildren(new Option('选择平台页面', ''));
    context.menus.forEach(menu => $('platformMenu').append(new Option(menu.name, menu.code)));
    $('businessPanel').hidden = false; $('connectionPanel').hidden = true;
    $('statusPill').textContent = context.user?.username || '已连接';
    restoreRoute(context.route || location.hash);
    await bridge.ready();
    await loadPosts();
    if (state.post) await openEditor(state.post);
  } catch (error) {
    state.online = false; $('businessPanel').hidden = true;
    $('connectionPanel').hidden = false;
    $('connectionText').textContent = error.message;
  } finally { $('connectBtn').disabled = false; }
}
$('connectBtn').addEventListener('click', connect);
window.addEventListener('beforeunload', event => { if (state.dirty) { event.preventDefault(); event.returnValue = ''; } });
void connect();
