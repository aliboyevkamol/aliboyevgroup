/* Admin panel. Talks to the Go backend (/api/v1/admin/*). The access token is kept in memory only;
   the refresh token is an HttpOnly cookie. Only users with role ADMIN can use this panel. */
(() => {
'use strict';
const $ = (s, r = document) => r.querySelector(s), $$ = (s, r = document) => [...r.querySelectorAll(s)];
const esc = s => String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const fmt = d => new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
const money = n => '$' + Number(n || 0).toLocaleString(undefined, { maximumFractionDigits: 2 });
const ini = n => String(n).split(/\s+/).map(w => w[0]).slice(0, 2).join('').toUpperCase();
const titleCase = s => s ? s[0] + s.slice(1).toLowerCase() : '';
const CATS = { websites: 'Websites', saas: 'SaaS', bots: 'Telegram Bots', ai: 'AI Tools', templates: 'Templates', automation: 'Automation' };
const BLOG = ['Development', 'AI', 'SaaS', 'Automation', 'Business', 'Tutorials'];
const STAT = ['Pending', 'Processing', 'Paid', 'Completed', 'Cancelled', 'Refunded'];

/* ---------- API client ---------- */
const ORIGIN = (() => {
 try { const o = localStorage.getItem('kamol_api_origin'); if (o) return o.replace(/\/$/, ''); } catch (e) {}
 const h = location.hostname;
 return location.protocol === 'file:' || ((h === 'localhost' || h === '127.0.0.1') && location.port !== '8080') ? 'http://localhost:8080' : '';
})();
const BASE = ORIGIN + '/api/v1';
let token = null, me = null;
const hint = on => { try { on === undefined ? null : (on ? sessionStorage.setItem('kamol_admin_hint', '1') : sessionStorage.removeItem('kamol_admin_hint')); return sessionStorage.getItem('kamol_admin_hint') === '1'; } catch (e) { return false; } };
async function req(method, path, body, retried) {
 const headers = {}; if (body !== undefined) headers['Content-Type'] = 'application/json'; if (token) headers.Authorization = 'Bearer ' + token;
 let r; try { r = await fetch(BASE + path, { method, headers, credentials: 'include', body: body === undefined ? undefined : JSON.stringify(body) }); }
 catch (e) { return { ok: false, status: 0, message: 'Cannot reach the server.' }; }
 if (r.status === 401 && !retried && path !== '/auth/refresh' && path !== '/auth/login' && (await refresh())) return req(method, path, body, true);
 if (r.status === 204) return { ok: true, data: null };
 let j = null; try { j = await r.json(); } catch (e) {}
 if (r.ok) return { ok: true, data: j && j.data, pagination: j && j.pagination };
 if (r.status === 401) { signedOut(); }
 return { ok: false, status: r.status, message: (j && j.message) || 'Something went wrong.', fields: j && j.error && j.error.fields };
}
async function refresh() { const r = await fetch(BASE + '/auth/refresh', { method: 'POST', credentials: 'include' }).catch(() => null); if (!r || !r.ok) { token = null; return false; } const j = await r.json(); token = j.data.access_token; me = j.data.user; return me.role === 'ADMIN'; }
const errText = r => r.fields ? Object.entries(r.fields).map(([k, v]) => k.replace(/_/g, ' ') + ' ' + v).join('; ') : r.message;

/* ---------- API <-> form shapes ---------- */
const S = {
 products: { path: '/admin/products', from: p => ({ id: p.id, title: p.title, category: p.category, description: p.description, price: Number(p.price), oldPrice: Number(p.old_price || 0), tech: p.technologies, demoUrl: p.demo_url, downloadUrl: p.download_url || '', featured: p.featured, published: p.published }),
  to: o => ({ title: o.title, category: o.category, description: o.description, price: o.price, old_price: o.oldPrice, technologies: o.tech, demo_url: o.demoUrl, download_url: o.downloadUrl, featured: o.featured, published: o.published }) },
 projects: { path: '/admin/projects', from: p => ({ id: p.id, title: p.title, cat: p.category, desc: p.description, tech: p.technologies, liveUrl: p.live_url, githubUrl: p.github_url, featured: p.featured, published: p.published }),
  to: o => ({ title: o.title, category: o.cat, description: o.desc, technologies: o.tech, live_url: o.liveUrl, github_url: o.githubUrl, featured: o.featured, published: o.published }) },
 posts: { path: '/admin/posts', from: p => ({ id: p.id, title: p.title, category: p.category, excerpt: p.excerpt, body: p.content, author: p.author, readingTime: p.reading_time, cover: p.cover_image || '', published: p.published }),
  to: o => ({ title: o.title, category: o.category, excerpt: o.excerpt, content: o.body, author: o.author, reading_time: o.readingTime, published: o.published }) }
};
const cache = { products: [], projects: [], posts: [] };
async function load(name) { const r = await req('GET', S[name].path + '?limit=100'); if (!r.ok) { toast(r.message); return false; } cache[name] = r.data.map(S[name].from); return true; }

/* ---------- UI helpers ---------- */
let tt; const toast = m => { const t = $('#toast'); t.textContent = m; t.classList.add('on'); clearTimeout(tt); tt = setTimeout(() => t.classList.remove('on'), 2600); };
function sheet(html, mount) {
 const w = document.createElement('div'); w.className = 'wrap'; w.innerHTML = `<div class="sheet" role="dialog" aria-modal="true">${html}</div>`; document.body.appendChild(w);
 requestAnimationFrame(() => w.classList.add('on'));
 const close = () => { w.classList.remove('on'); setTimeout(() => w.remove(), 220); document.removeEventListener('keydown', key); };
 const key = e => { if (e.key === 'Escape') close(); }; document.addEventListener('keydown', key);
 w.addEventListener('click', e => { if (e.target === w || e.target.closest('[data-close]')) close(); });
 mount && mount($('.sheet', w), close); const f = $('input,textarea,select,button', w); f && f.focus(); return close;
}
const confirmBox = (msg, ok) => sheet(`<h2>Are you sure?</h2><p class="mu">${esc(msg)}</p><div class="acts"><button class="btn" data-close>Cancel</button><button class="btn danger" data-ok>Delete</button></div>`, (s, close) => $('[data-ok]', s).addEventListener('click', async () => { await ok(); close(); }));
const sw = (attrs, on) => `<label class="sw"><input type="checkbox" ${attrs} ${on ? 'checked' : ''}><i></i></label>`;
const chips = (arr, cur, attr = 'data-f') => `<div class="chips" role="group">${arr.map(([k, l]) => `<button class="chip${k === cur ? ' on' : ''}" ${attr}="${k}">${l}</button>`).join('')}</div>`;

const E = {
 products: { name: 'Products', noun: 'product', sub: p => `${CATS[p.category] || p.category} · ${money(p.price)}`, blank: { category: 'websites', price: 0, published: true, tech: [] }, fields: [
  ['title', 'Product name', 'text'], ['category', 'Category', 'select', Object.entries(CATS)], ['description', 'Description', 'area'], ['price', 'Price', 'num', 0, 1], ['oldPrice', 'Old price', 'num', 0, 1],
  ['tech', 'Technologies (comma separated)', 'list'], ['demoUrl', 'Demo URL', 'text'], ['downloadUrl', 'Download URL or private file key', 'text'], ['featured', 'Featured', 'bool'], ['published', 'Published', 'bool']] },
 projects: { name: 'Projects', noun: 'project', sub: p => p.cat, blank: { published: true, tech: [] }, fields: [
  ['title', 'Project name', 'text'], ['cat', 'Category', 'text'], ['desc', 'Short description', 'area'], ['tech', 'Technologies (comma separated)', 'list'], ['liveUrl', 'Live URL', 'text'], ['githubUrl', 'GitHub URL', 'text'], ['featured', 'Featured', 'bool'], ['published', 'Published', 'bool']] },
 posts: { name: 'Blog', noun: 'post', sub: p => `${p.category} · ${p.readingTime || ''}`, blank: { category: 'Development', readingTime: '', published: true }, fields: [
  ['title', 'Title', 'text'], ['category', 'Category', 'select', BLOG.map(c => [c, c])], ['excerpt', 'Excerpt', 'area'], ['body', 'Content', 'area'], ['author', 'Author', 'text'], ['readingTime', 'Reading time (empty = automatic)', 'text'], ['published', 'Published', 'bool']] }
};
const inp = ([k, l, t, o], v) => {
 if (t === 'bool') return `<label class="chk">${l}${sw(`name="${k}"`, v)}</label>`;
 if (t === 'area') return `<label>${l}<textarea name="${k}" rows="${k === 'body' ? 6 : 3}">${esc(v)}</textarea></label>`;
 if (t === 'list') return `<label>${l}<input name="${k}" value="${esc((v || []).join(', '))}"></label>`;
 if (t === 'select') return `<label>${l}<select name="${k}">${o.map(([a, b]) => `<option value="${esc(a)}" ${a === v ? 'selected' : ''}>${esc(b)}</option>`).join('')}</select></label>`;
 return `<label>${l}<input name="${k}" type="${t === 'num' ? 'number' : 'text'}" ${t === 'num' ? 'inputmode="decimal" step="any" min="0"' : ''} value="${esc(v)}"></label>`;
};
function editor(name, id, after) {
 const e = E[name], cur = id ? cache[name].find(x => x.id === id) : { ...e.blank };
 sheet(`<h2>${id ? 'Edit' : 'New'} ${e.noun}</h2><form id="ef" class="fgrid two" novalidate>${e.fields.map(f => `<div ${f[4] ? '' : 'style="grid-column:1/-1"'}>${inp(f, cur[f[0]])}</div>`).join('')}<p class="err" id="ee" style="grid-column:1/-1"></p><div class="acts" style="grid-column:1/-1"><button class="btn pri" type="submit">Save</button><button class="btn" type="button" data-close>Cancel</button></div></form>`, (s, close) => {
  $('#ef', s).addEventListener('submit', async ev => {
   ev.preventDefault(); const o = { ...cur };
   e.fields.forEach(([k, , t]) => { const el = $(`[name="${k}"]`, s); o[k] = t === 'bool' ? el.checked : t === 'num' ? Number(el.value) || 0 : t === 'list' ? el.value.split(',').map(x => x.trim()).filter(Boolean) : el.value.trim(); });
   if (!o.title) { $('#ee', s).textContent = 'Title is required.'; return; }
   const btn = $('button[type=submit]', s); btn.disabled = true;
   const r = await req(id ? 'PATCH' : 'POST', S[name].path + (id ? '/' + id : ''), { ...S[name].to(o), cover_image: o.cover || null });
   if (!r.ok) { $('#ee', s).textContent = errText(r); btn.disabled = false; return; }
   close(); toast(id ? 'Changes saved' : 'Created'); after();
  });
 });
}

/* ---------- Views ---------- */
async function dashboard(v) {
 v.innerHTML = '<p class="empty">Loading…</p>';
 const r = await req('GET', '/admin/dashboard'); if (!r.ok) { v.innerHTML = `<p class="empty">${esc(r.message)}</p>`; return; }
 const d = r.data, days = d.revenue_7d.map(x => ({ l: new Date(x.date + 'T00:00:00').toLocaleDateString(undefined, { weekday: 'short' }), s: Number(x.total) })), mx = Math.max(1, ...days.map(x => x.s));
 const st = [['Revenue', money(d.total_revenue), 'paid orders'], ['Orders', d.total_orders, 'Pending: ' + d.pending_orders], ['Products', d.total_products, ''], ['Projects', d.total_projects, ''], ['Blog posts', d.total_posts, ''], ['Users', d.total_users, d.unread_messages + ' unread messages']];
 v.innerHTML = `<div class="page-h"><h2>Overview</h2></div><div class="stats">${st.map(([l, n, s]) => `<div class="card stat"><small>${l}</small><b>${n}</b><small class="up">${s}</small></div>`).join('')}</div>
 <div class="cols"><div class="card"><b>Revenue, last 7 days</b><div class="chart">${days.map(x => `<div><em>${x.s ? money(x.s) : ''}</em><i data-h="${Math.round(x.s / mx * 100)}"></i><small>${x.l}</small></div>`).join('')}</div></div>
 <div class="card"><b>Quick actions</b><div class="qa"><button class="btn pri sm" data-qa="products">+ Product</button><button class="btn sm" data-qa="projects">+ Project</button><button class="btn sm" data-qa="posts">+ Blog post</button></div>
 <b style="display:block;margin-top:20px">Recent comments</b>${d.recent_comments.slice(0, 3).map(c => `<div class="lrow"><span class="av">${esc(ini(c.name))}</span><div><p>${esc(c.content)}</p><small>${esc(c.name)}</small></div></div>`).join('') || '<p class="mu">No comments yet.</p>'}</div></div>
 <div class="card" style="margin-top:12px"><b>Recent orders</b>${d.recent_orders.map(o => `<div class="lrow"><div><p>#${esc(o.order_number)} · ${esc(o.customer_name)}</p><small>${fmt(o.created_at)}</small></div><b>${money(o.total)}</b><span class="badge ${titleCase(o.status)}">${titleCase(o.status)}</span></div>`).join('') || '<p class="mu">No orders yet.</p>'}</div>`;
 requestAnimationFrame(() => requestAnimationFrame(() => $$('.chart i', v).forEach(i => { i.style.height = Math.max(4, i.dataset.h) + '%'; })));
 v.onclick = e => { const b = e.target.closest('[data-qa]'); if (b) { const n = b.dataset.qa; load(n).then(() => editor(n, null, () => go(n))); } };
}

async function manage(v, name) {
 const e = E[name]; let q = '', f = 'all';
 v.innerHTML = '<p class="empty">Loading…</p>'; if (!(await load(name))) return;
 const flt = name === 'posts' ? [['all', 'All'], ['pub', 'Published'], ['draft', 'Drafts']] : [['all', 'All'], ['pub', 'Published'], ['draft', 'Drafts'], ['feat', 'Featured']];
 v.innerHTML = `<div class="page-h"><h2>${e.name}</h2><button class="btn pri" data-new>+ New ${e.noun}</button></div><div class="tool"><input type="search" id="q" placeholder="Search ${e.name.toLowerCase()}…" aria-label="Search"><div id="fc"></div></div><div class="items g2" id="il"></div>`;
 const draw = () => {
  $('#fc').innerHTML = chips(flt, f);
  const l = cache[name].filter(x => (!q || (x.title + ' ' + (x.cat || x.category || '')).toLowerCase().includes(q)) && (f === 'all' || (f === 'pub' && x.published) || (f === 'draft' && !x.published) || (f === 'feat' && x.featured)));
  $('#il').innerHTML = l.map(x => `<article class="card item" data-id="${x.id}"><div class="th">${esc(x.title[0] || '?')}</div><div><h3>${esc(x.title)}</h3><p>${esc(e.sub(x))}</p><span class="badge ${x.published ? 'ok' : 'Draft'}">${x.published ? 'Published' : 'Draft'}</span>${x.featured ? ' <span class="badge">Featured</span>' : ''}</div>${sw(`data-pub aria-label="Published"`, x.published)}<div class="acts"><button class="btn sm" data-edit>Edit</button><button class="btn sm" data-dup>Duplicate</button><button class="btn sm danger" data-del>Delete</button></div></article>`).join('') || `<p class="empty">Nothing found.</p>`;
 };
 const refresh = async () => { await load(name); draw(); };
 v.oninput = ev => { if (ev.target.id === 'q') { q = ev.target.value.toLowerCase(); draw(); } };
 v.onchange = async ev => {
  const c = ev.target.closest('[data-pub]'); if (!c) return;
  const r = await req('PATCH', S[name].path + '/' + c.closest('.item').dataset.id, { published: c.checked });
  toast(r.ok ? (c.checked ? 'Published' : 'Moved to drafts') : r.message); refresh();
 };
 v.onclick = async ev => {
  const b = ev.target.closest('button'); if (!b) return;
  if ('new' in b.dataset) return editor(name, null, refresh);
  if (b.dataset.f) { f = b.dataset.f; return draw(); }
  const it = b.closest('.item'); if (!it) return; const id = +it.dataset.id;
  if ('edit' in b.dataset) editor(name, id, refresh);
  else if ('dup' in b.dataset) { const c = { ...cache[name].find(x => x.id === id) }; c.title += ' (copy)'; c.published = false; c.featured = false; const r = await req('POST', S[name].path, S[name].to(c)); toast(r.ok ? 'Duplicated as draft' : errText(r)); refresh(); }
  else if ('del' in b.dataset) confirmBox('This item will be permanently deleted.', async () => { const r = await req('DELETE', S[name].path + '/' + id); toast(r.ok ? 'Deleted' : r.message); refresh(); });
 };
 draw();
}

async function orders(v) {
 let f = 'all', all = [];
 const draw = () => {
  const l = all.filter(o => f === 'all' || o.status === f);
  v.innerHTML = `<div class="page-h"><h2>Orders</h2></div><div class="tool">${chips([['all', 'All (' + all.length + ')'], ...STAT.map(s => [s, s])], f)}</div><div class="items">${l.map(o => `<article class="card item ord"><div><h3>#${esc(o.number)} · ${esc(o.customer)}</h3><p>${esc(o.email)} · ${fmt(o.date)}</p></div><b>${money(o.total)}</b><div class="its">${o.items.map(i => `${esc(i.title)} × ${i.quantity}`).join(' · ')}</div><div class="row"><span class="badge ${o.status}">${o.status}</span><select data-st="${o.id}" aria-label="Change status of order ${esc(o.number)}">${STAT.map(s => `<option ${s === o.status ? 'selected' : ''}>${s}</option>`).join('')}</select></div></article>`).join('') || '<p class="empty">No orders in this status.</p>'}</div>`;
 };
 v.innerHTML = '<p class="empty">Loading…</p>';
 const fetchAll = async () => { const r = await req('GET', '/admin/orders?limit=100'); if (!r.ok) { toast(r.message); return; } all = r.data.map(o => ({ id: o.id, number: o.order_number, customer: o.customer_name, email: o.customer_email, total: Number(o.total), date: o.created_at, status: titleCase(o.status), items: o.items })); draw(); };
 v.onclick = e => { const b = e.target.closest('[data-f]'); if (b) { f = b.dataset.f; draw(); } };
 v.onchange = async e => { const s = e.target.closest('[data-st]'); if (!s) return; const r = await req('PATCH', '/admin/orders/' + s.dataset.st, { status: s.value.toUpperCase() }); toast(r.ok ? 'Status: ' + s.value : r.message); fetchAll(); };
 fetchAll();
}

async function comments(v) {
 let f = 'all', all = [];
 const draw = () => {
  const l = all.filter(c => f === 'all' || (f === 'hidden') === c.hidden);
  v.innerHTML = `<div class="page-h"><h2>Comments</h2></div><div class="tool">${chips([['all', 'All'], ['visible', 'Visible'], ['hidden', 'Hidden']], f)}</div><div class="items">${l.map(c => `<article class="card item" data-id="${c.id}" style="grid-template-columns:auto 1fr"><span class="av">${esc(ini(c.name))}</span><div><h3>${esc(c.name)} <span class="badge ${c.hidden ? 'Hidden' : 'Visible'}">${c.hidden ? 'Hidden' : 'Visible'}</span></h3><p>${esc(c.text)}</p><small>${esc(c.on)} · ${fmt(c.date)}</small></div><div class="acts"><button class="btn sm" data-h>${c.hidden ? 'Restore' : 'Hide'}</button><button class="btn sm danger" data-x>Delete</button></div></article>`).join('') || '<p class="empty">No comments here.</p>'}</div>`;
 };
 v.innerHTML = '<p class="empty">Loading…</p>';
 const fetchAll = async () => { const r = await req('GET', '/admin/comments?limit=100'); if (!r.ok) { toast(r.message); return; } all = r.data.filter(c => c.status !== 'DELETED').map(c => ({ id: c.id, name: c.name, text: c.content, on: (c.type === 'post' ? 'Blog: ' : 'Project: ') + (c.target_title || ''), date: c.created_at, hidden: c.status === 'HIDDEN' })); draw(); };
 v.onclick = async e => {
  const b = e.target.closest('button'); if (!b) return; if (b.dataset.f) { f = b.dataset.f; return draw(); }
  const id = +b.closest('.item').dataset.id;
  if ('h' in b.dataset) { const c = all.find(x => x.id === id), r = await req('PATCH', '/admin/comments/' + id, { status: c.hidden ? 'VISIBLE' : 'HIDDEN' }); toast(r.ok ? (c.hidden ? 'Comment restored' : 'Comment hidden') : r.message); fetchAll(); }
  else if ('x' in b.dataset) confirmBox('This comment will be deleted.', async () => { const r = await req('DELETE', '/admin/comments/' + id); toast(r.ok ? 'Deleted' : r.message); fetchAll(); });
 };
 fetchAll();
}

async function users(v) {
 let q = '', all = [];
 v.innerHTML = '<div class="page-h"><h2>Users</h2></div><div class="tool"><input type="search" id="q" placeholder="Search users…" aria-label="Search users"></div><div class="items g2" id="ul"><p class="empty">Loading…</p></div>';
 const draw = () => { $('#ul').innerHTML = all.filter(u => (u.name + u.email).toLowerCase().includes(q)).map(u => `<article class="card item"><span class="av">${esc(ini(u.name))}</span><div><h3>${esc(u.name)}${u.role === 'ADMIN' ? ' <span class="badge">Admin</span>' : ''}</h3><p>${esc(u.email)}</p><small>Joined ${fmt(u.created)} · ${u.orders} orders</small></div><div style="text-align:right"><span class="badge ${u.status}">${u.status}</span><br>${u.id === me.id ? '' : sw(`data-u="${u.id}" aria-label="Account active"`, u.status === 'Active')}</div></article>`).join('') || '<p class="empty">No users found.</p>'; };
 const fetchAll = async () => { const r = await req('GET', '/admin/users?limit=100'); if (!r.ok) { toast(r.message); return; } all = r.data.map(u => ({ id: u.id, name: u.name, email: u.email, role: u.role, created: u.created_at, orders: u.orders, status: titleCase(u.status) })); draw(); };
 v.oninput = e => { if (e.target.id === 'q') { q = e.target.value.toLowerCase(); draw(); } };
 v.onchange = async e => { const c = e.target.closest('[data-u]'); if (!c) return; const r = await req('PATCH', '/admin/users/' + c.dataset.u, { status: c.checked ? 'ACTIVE' : 'BLOCKED' }); toast(r.ok ? (c.checked ? 'User activated' : 'User blocked') : r.message); fetchAll(); };
 fetchAll();
}

async function portfolio(v) {
 v.innerHTML = '<p class="empty">Loading…</p>';
 const r = await req('GET', '/admin/portfolio'); if (!r.ok) { v.innerHTML = `<p class="empty">${esc(r.message)}</p>`; return; }
 const d = r.data, L = d.social_links || {}, sk = (d.skills || []).map(k => [k.category, k.name, k.level]);
 const draw = () => {
  $('#sk').innerHTML = sk.map((x, i) => `<div class="skill"><span>${esc(x[1])} <small>${esc(x[0])}</small></span><b><span data-v="${i}">${x[2]}</span>% <button class="btn sm danger" data-rm="${i}" aria-label="Remove ${esc(x[1])}">✕</button></b><input type="range" min="0" max="100" step="5" value="${x[2]}" data-r="${i}" aria-label="${esc(x[1])} level"></div>`).join('') || '<p class="empty">No skills yet.</p>';
 };
 v.innerHTML = `<div class="page-h"><h2>Portfolio content</h2><button class="btn pri" id="sv">Save changes</button></div>
 <div class="card fgrid"><b>Hero</b><label>Hero title (wrap gradient words in [[ ]])<input id="ht" value="${esc(d.hero_title)}"></label><label>Hero description<textarea id="hl" rows="3">${esc(d.hero_description)}</textarea></label></div>
 <div class="card fgrid two" style="margin-top:12px"><b style="grid-column:1/-1">Social &amp; contact</b><label>GitHub<input id="lg" value="${esc(L.github)}"></label><label>LinkedIn<input id="ll" value="${esc(L.linkedin)}"></label><label>Telegram<input id="lt" value="${esc(L.telegram)}"></label><label>Email<input id="le" value="${esc((L.email || '').replace('mailto:', ''))}"></label></div>
 <div class="card" style="margin-top:12px"><b>Skills</b><div id="sk"></div><div class="fgrid two" style="margin-top:14px"><label>New skill<input id="sn" placeholder="e.g. TypeScript"></label><label>Category<select id="sc">${['frontend', 'backend', 'database', 'tools', 'ai', 'automation'].map(c => `<option>${c}</option>`).join('')}</select></label></div><button class="btn" id="sa" style="margin-top:12px">+ Add skill</button></div>`;
 v.oninput = e => { if (e.target.dataset.r) { sk[+e.target.dataset.r][2] = +e.target.value; $(`[data-v="${e.target.dataset.r}"]`).textContent = e.target.value; } };
 v.onclick = async e => {
  const b = e.target.closest('button'); if (!b) return;
  if (b.dataset.rm) { sk.splice(+b.dataset.rm, 1); draw(); }
  else if (b.id === 'sa') { const n = $('#sn').value.trim(); if (!n) return toast('Enter a skill name'); sk.push([$('#sc').value, n, 80]); $('#sn').value = ''; draw(); }
  else if (b.id === 'sv') {
   const em = $('#le').value.trim(), body = { hero_title: $('#ht').value, hero_description: $('#hl').value, social_links: { github: $('#lg').value.trim(), linkedin: $('#ll').value.trim(), telegram: $('#lt').value.trim(), email: em ? 'mailto:' + em : '' }, skills: sk.map(([category, name, level]) => ({ category, name, level })) };
   const res = await req('PUT', '/admin/portfolio', body); toast(res.ok ? 'Portfolio content saved' : errText(res));
  }
 };
 draw();
}

/* ---------- Router, shell, auth ---------- */
const R = { dashboard: ['Dashboard', '🏠', dashboard], products: ['Products', '📦', v => manage(v, 'products')], projects: ['Projects', '🧩', v => manage(v, 'projects')], posts: ['Blog', '✍️', v => manage(v, 'posts')], portfolio: ['Portfolio', '🎨', portfolio], orders: ['Orders', '🧾', orders], comments: ['Comments', '💬', comments], users: ['Users', '👥', users] };
const TABS = ['dashboard', 'products', 'orders', 'comments'];
$('#nav').innerHTML = Object.entries(R).map(([k, [t, i]]) => `<a class="nl" href="#${k}" data-r="${k}"><span>${i}</span>${t}</a>`).join('');
$('#tabs').innerHTML = TABS.map(k => `<button data-go="${k}" data-r="${k}"><span>${R[k][1]}</span>${R[k][0]}</button>`).join('') + '<button id="more"><span>☰</span>More</button>';
const setDrawer = o => { $('#side').classList.toggle('open', o); $('#scrim').hidden = !o; };
const go = k => { if ((location.hash.slice(1) || 'dashboard') === k) route(); else location.hash = k; };
function route() {
 const k = R[location.hash.slice(1)] ? location.hash.slice(1) : 'dashboard', v = $('#view'); v.onclick = v.oninput = v.onchange = null;
 $('#ttl').textContent = R[k][0]; $$('[data-r]').forEach(a => a.classList.toggle('on', a.dataset.r === k)); setDrawer(false);
 R[k][2](v); scrollTo(0, 0);
}
function show(on) { $('#login').hidden = on; $('#app').hidden = !on; if (on) route(); }
function signedOut() { token = null; me = null; hint(false); show(false); }
$('#lf').addEventListener('submit', async e => {
 e.preventDefault(); const b = $('#lf button[type=submit]'); b.disabled = true; $('#lerr').textContent = '';
 const r = await fetch(BASE + '/auth/login', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ email: $('#ae').value, password: $('#ap').value }) }).then(x => x.json()).catch(() => ({ success: false, message: 'Cannot reach the server.' }));
 b.disabled = false;
 if (!r.success) { $('#lerr').textContent = r.message; return; }
 if (r.data.user.role !== 'ADMIN') { await fetch(BASE + '/auth/logout', { method: 'POST', credentials: 'include' }).catch(() => {}); $('#lerr').textContent = 'This account does not have admin access.'; return; }
 token = r.data.access_token; me = r.data.user; hint(true); $('#ap').value = ''; show(true);
});
$('#out').addEventListener('click', async () => { await fetch(BASE + '/auth/logout', { method: 'POST', credentials: 'include' }).catch(() => {}); signedOut(); });
$('#menu').addEventListener('click', () => setDrawer(true)); $('#scrim').addEventListener('click', () => setDrawer(false)); $('#more').addEventListener('click', () => setDrawer(true));
$('#tabs').addEventListener('click', e => { const b = e.target.closest('[data-go]'); if (b) go(b.dataset.go); });
$('#thm').addEventListener('click', () => { const r = document.documentElement; r.dataset.theme = r.dataset.theme === 'dark' ? 'light' : 'dark'; });
addEventListener('hashchange', () => token && route());
(async () => { if (hint() && (await refresh())) show(true); else show(false); })();
})();
