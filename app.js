/* Kamol Aliboyev — shared script for every page. Detects the page via <body data-page>. */
(async () => {
'use strict';
const $ = (s, r = document) => r.querySelector(s), $$ = (s, r = document) => [...r.querySelectorAll(s)];
const page = document.body.dataset.page, qs = new URLSearchParams(location.search);
const reduce = matchMedia('(prefers-reduced-motion: reduce)').matches;
const fine = matchMedia('(hover: hover) and (pointer: fine)').matches;

/* ===================== DATA (edit these to change the site) ===================== */
const CAT = { websites: 'Websites', saas: 'SaaS', bots: 'Telegram Bots', ai: 'AI Tools', templates: 'Templates', automation: 'Automation' };

const SEED_PROJECTS = [
 { slug: 'digital-school', title: 'Digital School OS', cat: 'EdTech / SaaS', mock: 1, tech: ['React', 'Next.js', 'Supabase', 'PostgreSQL', 'AI'],
   desc: 'A digital education platform designed to manage learning, courses, students, progress and educational workflows.',
   problem: 'Learning centres juggle courses, students and progress across disconnected tools.',
   solution: 'One platform where courses, student progress and workflows live together, with AI assistance layered on top.',
   features: ['Course and lesson management', 'Student accounts and progress tracking', 'Role-based dashboards', 'AI-assisted learning workflows'],
   arch: ['Web App', 'API', 'Auth', 'PostgreSQL', 'AI'], outcome: 'A single workspace for managing learning end to end.' },
 { slug: 'ai-bosh-buxgalter', title: 'AI Bosh Buxgalter', cat: 'AI / Business Automation', mock: 2, tech: ['AI', 'APIs', 'Automation', 'PostgreSQL', 'Go'],
   desc: 'An AI-powered accounting assistant concept designed to automate accounting workflows and business financial operations.',
   problem: 'Routine accounting work is repetitive and slow for small businesses.',
   solution: 'An assistant that handles questions and routine bookkeeping steps through a chat interface backed by structured financial data.',
   features: ['Chat-based accounting assistant', 'Automated routine workflows', 'Structured ledger data', 'API integrations'],
   arch: ['Chat UI', 'Go API', 'AI Layer', 'PostgreSQL'], outcome: 'A concept showing how accounting workflows can be automated with AI.' },
 { slug: 'offpay', title: 'OffPay', cat: 'SaaS / Business', mock: 3, tech: ['JavaScript', 'Node.js', 'PostgreSQL', 'REST API'],
   desc: 'A business payment and invoice management platform.',
   problem: 'Businesses need one clear place to track invoices and payments.',
   solution: 'An admin platform with a dashboard for payments, invoices and reporting.',
   features: ['Invoice management', 'Payment tracking dashboard', 'Reporting views', 'REST API'],
   arch: ['Admin UI', 'REST API', 'Business Logic', 'PostgreSQL'], outcome: 'A SaaS-style admin panel for day-to-day payment operations.' },
 { slug: 'aliboyev-group', title: 'Aliboyev Group', cat: 'Business Platform', mock: 4, tech: ['HTML', 'CSS', 'JavaScript', 'Backend API'],
   desc: 'A modern business platform with product, service and digital infrastructure.',
   problem: 'A growing business needs a single, credible digital presence for products and services.',
   solution: 'A multi-page platform with a product catalogue and services, connected to a backend API.',
   features: ['Product and service pages', 'Responsive storefront', 'Backend API connection', 'Admin-managed content'],
   arch: ['Frontend', 'API', 'Database', 'Cloud'], outcome: 'A complete business platform from catalogue to backend.' },
 { slug: 'automation-systems', title: 'Automation Systems', cat: 'AI / Automation', mock: 5, tech: ['n8n', 'Make', 'Telegram Bot API', 'Google Sheets API'],
   desc: 'Workflow automation that connects bots, spreadsheets, APIs and AI into systems that run on their own.',
   problem: 'Teams lose hours on manual, repetitive steps between tools.',
   solution: 'Automated pipelines triggered by messages, forms or schedules that move data between services.',
   features: ['Telegram bot workflows', 'Spreadsheet sync', 'AI-powered steps', 'Scheduled jobs'],
   arch: ['Trigger', 'Workflow', 'AI Step', 'Output'], outcome: 'Repeatable automations that replace manual work.' }
];

// Example products with placeholder prices. Add, remove or edit entries; the store renders from this array.
const SEED_PRODUCTS = [
 { id: 1, title: 'Premium E-commerce Website', category: 'websites', price: 49, mock: 4, featured: true, technologies: ['HTML', 'CSS', 'JavaScript'], description: 'A fast, responsive online store front-end with catalogue, cart and checkout screens.', features: ['Product catalogue and filters', 'Cart and checkout UI', 'Fully responsive', 'Easy to restyle'] },
 { id: 2, title: 'Telegram Auto Service Bot', category: 'bots', price: 39, mock: 2, featured: true, technologies: ['Python', 'Telegram Bot API'], description: 'A bot that takes orders, answers common questions and notifies you of new requests.', features: ['Menu-driven ordering', 'Admin notifications', 'Multi-language ready', 'Simple config file'] },
 { id: 3, title: 'Admin Dashboard', category: 'templates', price: 59, mock: 3, technologies: ['HTML', 'CSS', 'JavaScript'], description: 'A dark dashboard template with charts, tables, filters and a collapsible sidebar.', features: ['Charts and KPI cards', 'Sortable tables', 'Dark theme', 'Mobile layout'] },
 { id: 4, title: 'AI Assistant', category: 'ai', price: 79, mock: 2, technologies: ['Node.js', 'AI APIs'], description: 'A chat assistant starter that connects to an AI provider and your own business data.', features: ['Chat interface', 'Provider-agnostic API layer', 'Conversation history', 'Prompt configuration'] },
 { id: 5, title: 'SaaS Starter', category: 'saas', price: 99, mock: 1, featured: true, technologies: ['Next.js', 'Supabase', 'PostgreSQL'], description: 'A starting point for a subscription product: auth, dashboard and account screens.', features: ['Authentication flow', 'Dashboard shell', 'Account settings', 'Database schema'] },
 { id: 6, title: 'Business Management System', category: 'saas', price: 129, mock: 3, technologies: ['Node.js', 'PostgreSQL', 'REST API'], description: 'Manage clients, invoices and tasks from one panel.', features: ['Client records', 'Invoices', 'Task board', 'REST API'] },
 { id: 7, title: 'Landing Page Pack', category: 'templates', price: 29, mock: 4, technologies: ['HTML', 'CSS'], description: 'Five conversion-focused landing page layouts in a consistent design system.', features: ['Five layouts', 'Pure HTML and CSS', 'Responsive', 'Dark and light themes'] },
 { id: 8, title: 'Workflow Automation Pack', category: 'automation', price: 45, mock: 5, technologies: ['n8n', 'Make', 'Google Sheets API'], description: 'Ready-made workflows for lead capture, notifications and spreadsheet sync.', features: ['Lead capture flow', 'Notification flow', 'Sheets sync', 'Setup notes'] }
];

// Placeholder articles. Replace with your own writing.
const SEED_POSTS = [
 { id: 1, title: 'How I structure a modern SaaS project', category: 'SaaS', date: 'Sep 2026', readingTime: '6 min', excerpt: 'A thin web app, a typed API, one database and clear boundaries. The structure I start every SaaS with.',
   body: '<p>Every SaaS I start follows the same shape: a thin web app, a clear API, one database and a boundary for anything slow.</p><h2>Start with the data model</h2><p>Before screens, I list the entities, who owns them and what a user can do with them. Most rework later traces back to an unclear ownership rule.</p><pre>users → workspaces → projects → items</pre><h2>Keep the API boring</h2><p>Plain REST endpoints, consistent errors and authentication handled in one place.</p>' },
 { id: 2, title: 'Building AI-powered business automation', category: 'AI', date: 'Aug 2026', readingTime: '7 min', excerpt: 'Where AI actually helps in a business workflow, and where a plain script is the better tool.',
   body: '<p>AI is best used for the fuzzy steps in a workflow: reading text, classifying requests, drafting replies.</p><h2>Keep the deterministic parts deterministic</h2><p>Calculations, permissions and payments should never depend on a model. Wrap the AI step so the rest of the pipeline stays predictable.</p><pre>trigger → validate → AI step → human check → save</pre><h2>Always leave a review step</h2><p>For anything touching money or customers, a person confirms the result before it is final.</p>' },
 { id: 3, title: 'Frontend architecture for large dashboards', category: 'Development', date: 'Jul 2026', readingTime: '8 min', excerpt: 'Keeping data-heavy interfaces fast and maintainable as screens and features pile up.',
   body: '<p>Dashboards grow quickly. The trick is to keep each screen small and the data flow obvious.</p><h2>One source of truth</h2><p>Keep state in one place per screen and derive everything else from it.</p><pre>const state = { filter: "all", sort: "new" };\nfunction render() { /* draw from state */ }</pre><h2>Render only what changed</h2><p>Large tables need pagination or virtualisation long before they feel slow.</p>' },
 { id: 4, title: 'Supabase vs PostgreSQL for SaaS', category: 'SaaS', date: 'Jun 2026', readingTime: '5 min', excerpt: 'Supabase is PostgreSQL with batteries included. When the extras help and when they get in the way.',
   body: '<p>Supabase is PostgreSQL plus authentication, storage and auto-generated APIs.</p><h2>When Supabase wins</h2><p>Early products where speed matters and the team is small.</p><h2>When plain PostgreSQL wins</h2><p>When you need full control over hosting, migrations and custom backend logic.</p><pre>SELECT * FROM projects WHERE owner_id = $1;</pre>' },
 { id: 5, title: 'How I build Telegram automation systems', category: 'Automation', date: 'May 2026', readingTime: '6 min', excerpt: 'From a simple menu bot to a full ordering and notification system.',
   body: '<p>A Telegram bot is a small web service that receives messages and replies. Treat it like any other backend.</p><h2>Design the conversation first</h2><p>Sketch the menu and every state before writing handlers.</p><pre>/start → menu → choose service → confirm → notify admin</pre><h2>Connect the rest</h2><p>Spreadsheets, databases and AI steps plug in behind the handlers.</p>' }
];

// [category, name, level] — visual indicators, not certifications.
const SEED_SKILLS = [
 ['frontend','HTML',92],['frontend','CSS',90],['frontend','JavaScript',92],['frontend','React',90],['frontend','Next.js',88],['frontend','Tailwind',85],['frontend','Vite',80],
 ['backend','Node.js',90],['backend','Python',85],['backend','Go',70],['backend','REST API',90],['backend','Prisma',80],['backend','Authentication',85],
 ['database','PostgreSQL',85],['database','Supabase',85],['database','Firebase',80],['database','SQL',85],
 ['tools','Git',90],['tools','GitHub',90],['tools','Docker',80],['tools','Linux',80],['tools','Vercel',85],['tools','AWS',70],['tools','DigitalOcean',75],['tools','Figma',75],
 ['ai','OpenAI',88],['ai','Claude',88],['ai','Gemini',85],['ai','AI APIs',88],['ai','AI Agents',80],['ai','Prompt Engineering',88],
 ['automation','n8n',85],['automation','Make',80],['automation','Telegram Bot API',90],['automation','Google Sheets API',80]
];

/* ===================== API CLIENT & STATE =====================
   The backend (Go + PostgreSQL) is the source of truth. The access token lives only in memory;
   the refresh token is an HttpOnly cookie that JavaScript cannot read. */

/* @api-client:begin */
/* ----- API URL builder: the single place where request URLs are made ----- */
const DEFAULT_PROD_ORIGIN = 'https://aliboyevgroup-backend.onrender.com';
const DEV_ORIGIN = 'http://localhost:8080';
const API_PREFIX = '/api/v1';
const normalizeOrigin = value => String(value || '').trim().replace(/\/+$/, '');
const isLocalHost = h => h === 'localhost' || h === '127.0.0.1' || h === '[::1]' || h === '::1';
const IS_DEV = location.protocol === 'file:' || isLocalHost(location.hostname);
// Turns a configured/stored value into a clean origin ("https://host[:port]"), or '' when it is unusable.
const cleanOrigin = value => {
  const s = normalizeOrigin(normalizeOrigin(value).replace(/\/api\/v1$/i, ''));
  if (!s) return '';
  try {
    const u = new URL(s);
    if (u.protocol !== 'https:' && u.protocol !== 'http:') return '';
    if (u.username || u.password) return '';
    if (u.protocol === 'http:' && !isLocalHost(u.hostname)) u.protocol = 'https:'; // no mixed content: plain http only for localhost
    if (!IS_DEV && isLocalHost(u.hostname)) return '';                              // a localhost origin on a production page is a stale dev leftover
    return u.origin;                                                                // drops any path, double slashes and trailing slashes
  } catch (e) { return ''; }
};
const DEFAULT_ORIGIN = IS_DEV ? DEV_ORIGIN : DEFAULT_PROD_ORIGIN;
let storedOrigin = '';
try {
  const raw = localStorage.getItem('kamol_api_origin');
  if (raw) { storedOrigin = cleanOrigin(raw); if (!storedOrigin) localStorage.removeItem('kamol_api_origin'); } // invalid value: remove it
} catch (e) {}
let API_ORIGIN = storedOrigin || DEFAULT_ORIGIN;
let API_BASE = API_ORIGIN + API_PREFIX;
const setApiOrigin = o => { API_ORIGIN = o; API_BASE = o + API_PREFIX; };
const apiUrl = path => API_BASE + '/' + String(path || '').replace(/^\/+/, '');

let accessToken = null;

const ls = {
  get: (k, d) => {
    try {
      const v = localStorage.getItem('kamol_' + k);
      return v ? JSON.parse(v) : d;
    } catch (e) {
      return d;
    }
  },
  set: (k, v) => {
    try {
      localStorage.setItem('kamol_' + k, JSON.stringify(v));
    } catch (e) {}
  }
};

// Non-secret hint that a session may exist (so guests skip the refresh call). Session-only unless "Remember me".
const hint = (on, persist) => { try { if (on === undefined) return localStorage.getItem('kamol_has_session') === '1' || sessionStorage.getItem('kamol_has_session') === '1'; localStorage.removeItem('kamol_has_session'); sessionStorage.removeItem('kamol_has_session'); if (on) (persist ? localStorage : sessionStorage).setItem('kamol_has_session', '1'); } catch (e) {} };
let refreshing = null;
const wait = ms => new Promise(r => setTimeout(r, ms));
const REQ_TIMEOUT = 45000; // Render free instances can need 30-50 s to wake up
const friendlyStatus = {
 400: 'The request was not valid. Please check your input.',
 401: 'Please log in to continue.',
 403: 'You do not have permission to do this.',
 404: 'Not found.',
 409: 'This conflicts with existing data. Please refresh and try again.',
 422: 'Some of the details are not valid. Please check and try again.',
 429: 'Too many requests. Please wait a moment and try again.',
 500: 'Something went wrong on our side. Please try again.',
 502: 'The server is temporarily unavailable. Please try again in a moment.',
 503: 'The server is temporarily unavailable. Please try again in a moment.',
 504: 'The server is temporarily unavailable. Please try again in a moment.'
};
const API = {
 // One network attempt. Never throws: returns { r, j } or { err: 'NETWORK' | 'TIMEOUT' }.
 async _fetch(method, path, body, opt) {
  const headers = {};
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (accessToken && !opt.noAuth) headers.Authorization = 'Bearer ' + accessToken;
  if (opt.headers) Object.assign(headers, opt.headers);
  const ctl = typeof AbortController === 'function' ? new AbortController() : null;
  const timer = ctl ? setTimeout(() => ctl.abort(), opt.timeout || REQ_TIMEOUT) : null;
  try {
   const r = await fetch(apiUrl(path), { method, headers, credentials: 'include', body: body === undefined ? undefined : JSON.stringify(body), signal: ctl ? ctl.signal : undefined });
   let j = null; if (r.status !== 204) { try { j = await r.json(); } catch (e) {} } // non-JSON bodies (HTML error pages) must not crash the app
   return { r, j };
  } catch (e) { return { err: e && e.name === 'AbortError' ? 'TIMEOUT' : 'NETWORK' }; }
  finally { if (timer) clearTimeout(timer); }
 },
 async req(method, path, body, opt = {}) {
  let x = await API._fetch(method, path, body, opt);
  // Only GET is repeated (once) on a network error or a gateway error while Render wakes up. POST/PUT/PATCH/DELETE are never repeated automatically.
  if (method === 'GET' && !opt.noRetry && (x.err === 'NETWORK' || (x.r && (x.r.status === 502 || x.r.status === 503 || x.r.status === 504)))) { await wait(1500); x = await API._fetch(method, path, body, opt); }
  if (x.err === 'TIMEOUT') return { ok: false, status: 0, code: 'TIMEOUT', message: method === 'GET' ? 'The server is taking too long to respond (it may be waking up). Please try again in a moment.' : 'The server is taking too long to respond. Please check before trying again: the action may already have gone through.' };
  if (x.err) return { ok: false, status: 0, code: 'NETWORK', message: 'Cannot reach the server. Please check your connection.' };
  const r = x.r, j = x.j;
  if (r.status === 401 && !opt.noAuth && !opt.retried && hint() && (await API.refresh())) return API.req(method, path, body, { ...opt, retried: true });
  if (r.status === 204) return { ok: true, status: 204, data: null };
  if (r.ok) {
   if (j === null || typeof j !== 'object') return { ok: false, status: r.status, code: 'BAD_RESPONSE', message: 'The server sent an unexpected response. Please try again later.' };
   return { ok: true, status: r.status, data: 'data' in j ? j.data : j, pagination: j.pagination };
  }
  const e = j && typeof j === 'object' ? j.error : null;
  const message = (j && typeof j === 'object' && typeof j.message === 'string' && j.message) || (e && typeof e === 'object' && typeof e.message === 'string' && e.message) || (typeof e === 'string' && e) || friendlyStatus[r.status] || (r.status >= 500 ? friendlyStatus[500] : 'Something went wrong.');
  return { ok: false, status: r.status, code: e && e.code, fields: e && e.fields, message };
 },
 get: p => API.req('GET', p), post: (p, b, o) => API.req('POST', p, b === undefined ? {} : b, o), del: p => API.req('DELETE', p),
 // Concurrent 401s share one refresh call (refresh tokens may rotate, so two parallel refreshes could invalidate each other).
 refresh() {
  if (!refreshing) refreshing = (async () => {
   const r = await API.req('POST', '/auth/refresh', undefined, { noAuth: true, retried: true });
   if (r.ok && r.data && r.data.access_token) { accessToken = r.data.access_token; return r.data.user || true; }
   accessToken = null;
   if (r.status === 400 || r.status === 401 || r.status === 403) hint(false); // forget the session only when the server rejected it, not on a network blip
   return null;
  })().finally(() => { refreshing = null; });
  return refreshing;
 }
};
const errText = r => { if (r.fields && typeof r.fields === 'object') { const f = Object.entries(r.fields)[0]; if (f) return f[0].replace(/^customer\./, '').replace(/_/g, ' ') + ' ' + f[1]; } return r.message; };
/* @api-client:end */

const esc = s => String(s == null ? '' : s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const ini = n => String(n).split(/\s+/).map(w => w[0]).slice(0, 2).join('').toUpperCase();
const fmt = d => new Date(d).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
const titleCase = s => s ? s[0] + s.slice(1).toLowerCase() : '';
// DOM-safety helpers: a missing element on some HTML page must never stop the whole script.
const out = sel => $(sel) || document.createElement('div');            // assignment target that is harmless when the element is absent
const val = sel => { const e = $(sel); return e && e.value != null ? e.value : ''; };
const on = (el, ev, fn, o) => { if (el) el.addEventListener(ev, fn, o); };
// URLs coming from the API: only http(s)/mailto (and optionally relative) are allowed, never javascript:/data:
const safeUrl = (u, rel) => { const s = String(u == null ? '' : u).trim(); if (/^(https?:|mailto:)/i.test(s)) return s; if (rel && s && /^(?![a-z][a-z0-9+.-]*:)(?![\/\\]{2})/i.test(s)) return s; return ''; };
// Small allow-list sanitizer for HTML that comes from the API (blog body, product long description).
const SAFE_TAGS = new Set(['p', 'br', 'strong', 'b', 'em', 'i', 'u', 's', 'a', 'ul', 'ol', 'li', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'blockquote', 'code', 'pre', 'hr', 'img', 'figure', 'figcaption', 'table', 'thead', 'tbody', 'tr', 'th', 'td', 'span', 'div', 'small', 'sup', 'sub', 'details', 'summary']);
const DROP_TAGS = new Set(['script', 'style', 'iframe', 'object', 'embed', 'link', 'meta', 'svg', 'math', 'form', 'input', 'button', 'textarea', 'select', 'base', 'template', 'noscript']);
const sanitizeHtml = html => {
 const doc = new DOMParser().parseFromString('<body>' + String(html == null ? '' : html) + '</body>', 'text/html');
 const walk = node => [...node.childNodes].forEach(ch => {
  if (ch.nodeType === 3) return;
  if (ch.nodeType !== 1) { ch.remove(); return; }
  const tag = ch.tagName.toLowerCase();
  if (DROP_TAGS.has(tag)) { ch.remove(); return; }
  if (!SAFE_TAGS.has(tag)) { walk(ch); ch.replaceWith(...ch.childNodes); return; }
  [...ch.attributes].forEach(a => {
   const n = a.name.toLowerCase();
   if (n === 'href' && tag === 'a') { const u = safeUrl(a.value, true); if (u) { ch.setAttribute('href', u); ch.setAttribute('rel', 'noopener noreferrer'); } else ch.removeAttribute('href'); }
   else if (n === 'src' && tag === 'img') { const u = safeUrl(a.value, true); if (u) ch.setAttribute('src', u); else ch.removeAttribute('src'); }
   else if (!['alt', 'title', 'class', 'width', 'height', 'colspan', 'rowspan'].includes(n)) ch.removeAttribute(a.name);
  });
  walk(ch);
 });
 walk(doc.body); return doc.body.innerHTML;
};

// API (snake_case) -> the shapes the existing pages already use
const mapProduct = p => ({ id: p.id, slug: p.slug, title: p.title, category: p.category, description: p.description, longDescription: p.content, price: Number(p.price), oldPrice: p.old_price ? Number(p.old_price) : 0, technologies: p.technologies || [], features: p.features || [], demoUrl: p.demo_url, demoType: p.demo_type, liveUrl: p.live_url, image: p.cover_image, hasDownload: p.has_download, featured: p.featured, mock: p.mock_variant || 1 });
const mapProject = p => ({ id: p.id, slug: p.slug, title: p.title, cat: p.category, desc: p.description, fullDesc: p.full_description, problem: p.problem, solution: p.solution, outcome: p.outcome, features: p.features || [], tech: p.technologies || [], arch: p.architecture && p.architecture.length ? p.architecture : null, demoUrl: p.demo_url, liveUrl: p.live_url, githubUrl: p.github_url, image: p.cover_image, featured: p.featured, mock: p.mock_variant || 1, likes: p.like_count, liked: p.liked });
const mapPost = p => ({ id: p.id, slug: p.slug, title: p.title, category: p.category, date: new Date(p.published_at || p.created_at).toLocaleDateString(undefined, { month: 'short', year: 'numeric' }), readingTime: p.reading_time, excerpt: p.excerpt, body: p.content, cover: p.cover_image, author: p.author, likes: p.like_count, liked: p.liked });
const normOrder = o => ({ id: o.id, number: o.order_number, status: o.status, date: o.created_at, total: Number(o.total), items: (o.items || []).map(i => ({ id: i.product_id, title: i.title, price: Number(i.unit_price), qty: i.quantity })) });

/* ----- initial load: session, content, cart, favorites ----- */
const state = { user: null, cart: (c => Array.isArray(c) ? c : [])(ls.get('cart', [])), favs: [], offline: false };
// The first request to a sleeping Render instance can take up to ~1 minute: tell the visitor instead of showing a blank page.
const wakeTimer = setTimeout(() => { try { if (!document.body) return; const t = $('.toast') || document.body.appendChild(Object.assign(document.createElement('div'), { className: 'toast', role: 'status' })); t.textContent = 'Connecting to the server\u2026 it may be waking up, this can take up to a minute.'; t.classList.add('on'); } catch (e) {} }, 4000);
const boot = await (async () => {
 const loadSession = async () => { let me = null; if (hint() && (await API.refresh())) { const m = await API.get('/auth/me'); if (m.ok && m.data) me = { ...m.data, created: m.data.created_at }; } return me; };
 const loadAll = me => Promise.all([API.get('/products?limit=100'), API.get('/projects?limit=50'), API.get('/posts?limit=100'), API.get('/portfolio'), me ? API.get('/cart') : null, me ? API.get('/me/favorites') : null]);
 const failed = x => !x.ok && (x.status === 0 || x.status === 404 || x.code === 'BAD_RESPONSE');
 let me = await loadSession(), res = await loadAll(me);
 // A stale/wrong kamol_api_origin must not break the site: if none of the core endpoints work on it, drop it and retry once on the default origin.
 if (API_ORIGIN !== DEFAULT_ORIGIN && res.slice(0, 3).every(failed)) {
  try { localStorage.removeItem('kamol_api_origin'); } catch (e) {}
  setApiOrigin(DEFAULT_ORIGIN);
  me = await loadSession(); res = await loadAll(me);
 }
 const [pr, pj, po, pf, cart, favs] = res;
 const okList = x => !!(x && x.ok && Array.isArray(x.data));
 const okPr = okList(pr), okPj = okList(pj), okPo = okList(po), okPf = !!(pf.ok && pf.data && typeof pf.data === 'object' && !Array.isArray(pf.data));
 // "offline" = none of the core content could be loaded; a single failing endpoint only falls back for that section.
 const offline = !okPr && !okPj && !okPo;
 return { me, offline, partial: !offline && !(okPr && okPj && okPo), okPr, okPj, okPo, okPf, cart, favs, pr, pj, po, pf };
})();
clearTimeout(wakeTimer); { const wt = $('.toast'); if (wt) wt.classList.remove('on'); }
state.user = boot.me; state.offline = boot.offline;
if (state.user && boot.cart && boot.cart.ok && boot.cart.data && Array.isArray(boot.cart.data.items)) state.cart = boot.cart.data.items.map(i => ({ id: i.product_id, qty: i.quantity }));
if (state.user && boot.favs && boot.favs.ok && Array.isArray(boot.favs.data)) state.favs = boot.favs.data.map(p => p.id);
const portfolioData = boot.okPf ? boot.pf.data : null;
const site = portfolioData ? { heroTitle: portfolioData.hero_title, heroLead: portfolioData.hero_description, links: portfolioData.social_links || {}, skills: (portfolioData.skills || []).map(k => [k.category, k.name, k.level]) } : { links: {}, skills: null };
// Sections that could not be loaded fall back to the bundled sample content (read-only) so the pages still render.
const projects = boot.okPj ? boot.pj.data.map(mapProject) : SEED_PROJECTS.map((p, i) => ({ mock: (i % 5) + 1, ...p }));
const products = boot.okPr ? boot.pr.data.map(mapProduct) : SEED_PRODUCTS.map(p => ({ mock: 1, features: [], technologies: [], ...p }));
const posts = boot.okPo ? boot.po.data.map(mapPost) : SEED_POSTS;
const skills = site.skills && site.skills.length ? site.skills : SEED_SKILLS;

/* ----- Cart: instant local state, mirrored to the server for logged-in users ----- */
const Cart = {
 raw: () => state.cart,
 _after() { updateBadge(); if (!state.user) ls.set('cart', state.cart); },
 // Server calls are made only for signed-in users; guests keep their cart in localStorage.
 _srv(fn) { if (!state.user) return; fn().then(r => { if (!r.ok) { toast(r.message); Cart.sync(); } }); },
 async sync() { const r = await API.get('/cart'); if (r.ok && r.data && Array.isArray(r.data.items)) { state.cart = r.data.items.map(i => ({ id: i.product_id, qty: i.quantity })); updateBadge(); } },
 add(id) { const x = state.cart.find(i => i.id == id); x ? x.qty++ : state.cart.push({ id: +id, qty: 1 }); this._after(); this._srv(() => API.post('/cart/items', { product_id: +id, quantity: 1 })); },
 remove(id) { state.cart = state.cart.filter(i => i.id != id); this._after(); this._srv(() => API.del('/cart/items/' + id)); },
 increase(id) { this.add(id); },
 decrease(id) { const x = state.cart.find(i => i.id == id); if (!x) return; if (x.qty <= 1) return this.remove(id); x.qty--; this._after(); this._srv(() => API.req('PATCH', '/cart/items/' + id, { quantity: x.qty })); },
 clear() { state.cart = []; this._after(); this._srv(() => API.del('/cart')); },
 getItems() { return state.cart.map(i => ({ ...i, product: products.find(p => p.id == i.id) })).filter(i => i.product); },
 getTotal() { return this.getItems().reduce((s, i) => s + i.product.price * i.qty, 0); },
 getCount() { return this.getItems().reduce((s, i) => s + i.qty, 0); },
 async mergeGuest() { const g0 = ls.get('cart', []), g = Array.isArray(g0) ? g0 : []; for (const i of g) await API.post('/cart/items', { product_id: i.id, quantity: i.qty }); ls.set('cart', []); await this.sync(); }
};

/* ----- Auth ----- */
const Auth = {
 currentUser: () => state.user,
 isLoggedIn: () => !!state.user,
 async _done(r, remember) {
  if (!r.ok) return { ok: false, error: errText(r) };
  if (!r.data || !r.data.access_token || !r.data.user) return { ok: false, error: 'The server sent an unexpected response. Please try again.' };
  accessToken = r.data.access_token; hint(true, remember); state.user = r.data.user; await Cart.mergeGuest();
  return { ok: true, user: state.user };
 },
 async register(f) { return this._done(await API.post('/auth/register', { name: f.name, email: f.email, password: f.password, confirm_password: f.confirm }, { noAuth: true }), true); },
 async login(email, password, remember) { return this._done(await API.post('/auth/login', { email, password }, { noAuth: true }), remember); },
 async logout() { await API.post('/auth/logout'); accessToken = null; hint(false); state.user = null; }
};

/* ----- Favorites (logged-in users; stored in PostgreSQL) ----- */
const Favs = {
 list: () => state.favs,
 has: id => state.favs.includes(+id),
 toggle(id) {
  id = +id; const on = !state.favs.includes(id);
  state.favs = on ? [...state.favs, id] : state.favs.filter(x => x !== id);
  API.req(on ? 'POST' : 'DELETE', '/products/' + id + '/favorite', on ? {} : undefined).then(r => { if (!r.ok) { toast(r.message); state.favs = on ? state.favs.filter(x => x !== id) : [...state.favs, id]; } });
  return on;
 }
};

/* ===================== HELPERS ===================== */
const M = {
 1: '<div class="side"><i></i><i></i><i></i><i></i></div><div class="mb"><div class="mh"><b></b><b></b></div><div class="course"><u></u><u></u><u></u></div><div class="prog"><s style="width:78%"></s></div><div class="prog"><s style="width:52%"></s></div></div>',
 2: '<div class="chat"><p class="u"></p><p class="a"></p><p class="u s"></p></div><div class="ledger"><i></i><i></i><i></i><i></i></div>',
 3: '<div class="kpis"><u></u><u></u><u></u></div><svg viewBox="0 0 200 60" preserveAspectRatio="none"><path d="M0 50L25 38 50 44 80 22 110 30 140 12 170 18 200 4" fill="none" stroke="#7C5CFF" stroke-width="2"/></svg><div class="rows"><i></i><i></i><i></i></div>',
 4: '<div class="pg"><u></u><u></u><u></u><u></u><u></u><u></u></div>',
 5: '<i></i><i></i><i></i>'
};
const mock = n => `<div class="mock m${n}" aria-hidden="true">${M[n] || M[1]}</div>`;
const chips = t => `<ul class="chips">${t.map(x => `<li>${esc(x)}</li>`).join('')}</ul>`;
const card = o => `<article class="card pc"><a class="pv" href="${o.href}" aria-label="${esc(o.t)}">${o.vis || mock(o.m)}</a><div class="pb"><small>${esc(o.k)}</small><h3><a href="${o.href}">${esc(o.t)}</a></h3><p>${esc(o.d)}</p>${chips(o.tech)}<div class="pf">${o.foot}</div></div></article>`;
const projCard = p => card({ href: `project-detail.html?project=${encodeURIComponent(p.slug)}`, m: p.mock, vis: p.image ? vis(p) : '', k: p.cat, t: p.title, d: p.desc, tech: p.tech, foot: `<a class="btn sm" href="project-detail.html?project=${encodeURIComponent(p.slug)}">View Case Study</a><a class="btn sm" href="#" data-demo>Live Demo ↗</a>` });
const prodCard = p => card({ href: `product.html?id=${p.id}`, m: p.mock, vis: p.image ? vis(p) : '', k: CAT[p.category] || p.category, t: p.title, d: p.description, tech: p.technologies, foot: `<b>$${p.price}${p.oldPrice ? ` <s class="old">$${p.oldPrice}</s>` : ''}</b><button class="ic fav" data-fav data-id="${p.id}" aria-pressed="${Favs.has(p.id)}" aria-label="Toggle favorite">${Favs.has(p.id) ? '♥' : '♡'}</button><div class="pa"><a class="btn sm" href="product.html?id=${p.id}">View Product</a><button class="btn sm" data-xdemo data-id="${p.id}">Demo</button><button class="btn sm" data-cart data-id="${p.id}">Add to Cart</button><button class="btn sm pri" data-buy data-id="${p.id}">Buy Now</button></div>` });
const postCard = p => `<article class="card bc"><small>${esc(p.category)} · ${esc(p.date)} · ${esc(p.readingTime)}</small><h3>${esc(p.title)}</h3><p>${esc(p.excerpt)}</p><a class="rd" href="article.html?id=${p.id}">Read Article →</a></article>`;
let tt; const toast = msg => { let t = $('.toast') || document.body.appendChild(Object.assign(document.createElement('div'), { className: 'toast', role: 'status' })); t.textContent = msg; t.classList.add('on'); clearTimeout(tt); tt = setTimeout(() => t.classList.remove('on'), 2600); };
const makePills = (el, items, cb) => {
 if (!el) return;
 el.innerHTML = items.map(([k, l], i) => `<button class="pill${i ? '' : ' on'}" data-k="${k}" aria-pressed="${!i}">${l}</button>`).join('');
 el.addEventListener('click', e => { const b = e.target.closest('.pill'); if (!b) return; $$('.pill', el).forEach(x => { x.classList.toggle('on', x === b); x.setAttribute('aria-pressed', x === b); }); cb(b.dataset.k); });
};

/* ===================== SHARED HEADER / FOOTER ===================== */
const NAV = [['index', 'Home'], ['about', 'About'], ['skills', 'Skills'], ['projects', 'Projects'], ['store', 'Store'], ['blog', 'Blog'], ['contact', 'Contact']];
const LANG = { EN: ['Home', 'About', 'Skills', 'Projects', 'Store', 'Blog', 'Contact', "Let's Talk"], UZ: ['Bosh sahifa', 'Men haqimda', "Ko'nikmalar", 'Loyihalar', "Do'kon", 'Blog', 'Aloqa', 'Bog‘lanish'], RU: ['Главная', 'Обо мне', 'Навыки', 'Проекты', 'Магазин', 'Блог', 'Контакты', 'Связаться'] };
const cur = { 'project-detail': 'projects', product: 'store', article: 'blog' }[page] || page;
const root = document.documentElement;
const store = { get: k => { try { return localStorage.getItem(k); } catch (e) { return null; } }, set: (k, v) => { try { localStorage.setItem(k, v); } catch (e) {} } };
root.dataset.theme = store.get('ka-theme') || 'dark';
let lang = store.get('ka-lang'); if (!Object.prototype.hasOwnProperty.call(LANG, lang)) lang = 'EN';

const header = document.createElement('header');
header.className = 'nav'; header.id = 'nav';
header.innerHTML = `<div class="wrap nav-in"><a href="index.html" class="brand" aria-label="Aliboyev Group, home"><img class="brand-logo" src="assets/logo-360.png" alt="ALIBOYEV GROUP" width="360" height="123" decoding="async"></a>
<nav class="links" id="links" aria-label="Main">${NAV.map(([p, l]) => `<a href="${p}.html" data-p="${p}"${p === cur ? ' class="act" aria-current="page"' : ''}>${l}</a>`).join('')}</nav>
<div class="tools"><button class="ic" id="theme" aria-label="Toggle theme">◐</button><button class="ic" id="lang" aria-label="Change language">${lang}</button><a class="btn sm" href="contact.html" id="talk">Let's Talk</a>
<button class="ic burger" id="burger" aria-label="Menu" aria-expanded="false" aria-controls="links"><i></i><i></i></button></div></div>`;
const footer = document.createElement('footer');
footer.className = 'wrap foot';
footer.innerHTML = `<div><a class="brand" href="index.html" aria-label="Aliboyev Group, home"><img class="brand-logo" src="assets/logo-360.png" alt="ALIBOYEV GROUP" width="360" height="123" decoding="async"></a><span>Kamol Aliboyev<br><small>Full-Stack Developer &amp; AI Product Builder</small></span></div>
<nav aria-label="Footer"><a href="index.html">Home</a><a href="about.html">About</a><a href="projects.html">Projects</a><a href="store.html">Store</a><a href="blog.html">Blog</a><a href="contact.html">Contact</a></nav>
<nav aria-label="Social"><a href="https://github.com/your-username">GitHub</a><a href="https://linkedin.com/in/your-username">LinkedIn</a><a href="https://t.me/your_username">Telegram</a></nav><small>© 2026 Kamol Aliboyev</small>`;
const main = $('#main'); if (main) { document.body.insertBefore(header, main); main.after(footer); } else { document.body.prepend(header); document.body.append(footer); } decorateHeader(); applySite(); if (state.offline) setTimeout(() => toast('Server unreachable: showing sample content. Sign in, cart and orders are unavailable.'), 400);
else if (boot.partial) setTimeout(() => toast('Some content could not be loaded from the server. Sample content is shown for those sections.'), 400);

const applyLang = () => { const L = LANG[lang]; $$('.links a').forEach((a, i) => a.textContent = L[i]); out('#talk').textContent = L[7]; out('#lang').textContent = lang; document.documentElement.lang = lang.toLowerCase(); };
applyLang();
on($('#theme'), 'click', () => { root.dataset.theme = root.dataset.theme === 'dark' ? 'light' : 'dark'; store.set('ka-theme', root.dataset.theme); });
on($('#lang'), 'click', () => { const k = Object.keys(LANG); lang = k[(k.indexOf(lang) + 1) % 3]; store.set('ka-lang', lang); applyLang(); });
const links = $('#links'), burger = $('#burger');
const setMenu = o => { links.classList.toggle('open', o); burger.setAttribute('aria-expanded', o); };
burger.addEventListener('click', () => setMenu(!links.classList.contains('open')));
addEventListener('keydown', e => e.key === 'Escape' && setMenu(false));
const onScroll = () => header.classList.toggle('on', scrollY > 20);
addEventListener('scroll', onScroll, { passive: true }); onScroll();

/* ===================== PAGES ===================== */
function initHome() {
 const fp = projects.filter(p => p.featured); out('#fp').innerHTML = (fp.length ? fp : projects).slice(0, 3).map(projCard).join('');
 out('#fs').innerHTML = products.filter(p => p.featured).slice(0, 3).map(prodCard).join('');
 const el = $('#code');
 const src = [['k', 'const'], ' developer = {\n  name: ', ['s', '"Kamol Aliboyev"'], ',\n  role: ', ['s', '"Full-Stack Developer"'], ',\n  focus: [', ['s', '"SaaS"'], ', ', ['s', '"AI"'], ', ', ['s', '"Automation"'], '],\n  status: ', ['s', '"available"'], '\n};'];
 const len = src.reduce((n, p) => n + (Array.isArray(p) ? p[1] : p).length, 0);
 const draw = n => { let left = n, h = ''; for (const p of src) { const t = Array.isArray(p) ? p[1] : p, part = t.slice(0, left); h += Array.isArray(p) ? `<span class="${p[0]}">${part}</span>` : part; left -= t.length; if (left <= 0) break; } if (el) el.innerHTML = h + '<span class="c"></span>'; };
 if (reduce) draw(len); else { let n = 0; (function t() { draw(n += 2); if (n < len) setTimeout(t, 24); })(); }
 if (!reduce && matchMedia('(min-width:1025px)').matches) addEventListener('scroll', () => { out('#term').style.transform = `translateY(${Math.min(scrollY * -.05, 0)}px)`; }, { passive: true });
}

function initSkills() {
 out('#skg').innerHTML = skills.map(([c, n, v]) => `<div class="sk" data-c="${esc(c)}"><div class="sr2"><span>${esc(n)}</span><b>${esc(v)}%</b></div><div class="bar2"><s data-w="${esc(v)}"></s></div></div>`).join('');
 const bars = new IntersectionObserver(es => es.forEach(e => { if (e.isIntersecting) { const s = $('s', e.target); s.style.width = s.dataset.w + '%'; bars.unobserve(e.target); } }), { threshold: .3 });
 $$('.bar2').forEach(b => bars.observe(b));
 makePills($('#pills'), [['all', 'All'], ['backend', 'Backend'], ['frontend', 'Frontend'], ['database', 'Database'], ['tools', 'Tools'], ['ai', 'AI'], ['automation', 'Automation']], k => {
  $$('.sk').forEach(el => {
   const show = k === 'all' || el.dataset.c === k, s = $('s', el);
   if (show) { el.hidden = false; s.style.width = '0'; requestAnimationFrame(() => requestAnimationFrame(() => { el.classList.remove('off'); s.style.width = s.dataset.w + '%'; })); }
   else { el.classList.add('off'); setTimeout(() => el.classList.contains('off') && (el.hidden = true), reduce ? 0 : 250); }
  });
 });
}

function initProjects() {
 out('#pl').innerHTML = projects.map((p, i) => `<article class="card proj${i % 2 ? ' rev' : ''}"><div class="pi"><span class="pn">0${i + 1}</span><small>${esc(p.cat)}</small><h3>${esc(p.title)}</h3><p>${esc(p.desc)}</p>${chips(p.tech)}<div class="pl"><a href="project-detail.html?project=${encodeURIComponent(p.slug)}">View Case Study <span>→</span></a><a href="#" data-demo>Live Demo <span>↗</span></a></div></div>${vis(p)}</article>`).join('');
}

function initDetail() {
 const i = Math.max(0, projects.findIndex(p => p.slug === qs.get('project'))), p = projects[i], nx = projects[(i + 1) % projects.length];
 if (!p) { out('#pd').innerHTML = '<p class="lead">Project not found.</p>'; return; }
 document.title = `${p.title} — Case study | Kamol Aliboyev`;
 out('#pd').innerHTML = `<a class="back" href="projects.html">← All projects</a><p class="lab">Case study / ${esc(p.cat)}</p><h1>${esc(p.title)}</h1><p class="lead">${esc(p.desc)}</p>${chips(p.tech)}<div class="big">${vis(p)}</div>
 <div class="cs"><h2>Overview</h2><p class="lead">${esc(p.fullDesc || p.desc)}</p><h2>Problem</h2><p class="lead">${esc(p.problem || '')}</p><h2>Solution</h2><p class="lead">${esc(p.solution || '')}</p>
 <h2>Features</h2><ul class="ul">${(p.features || []).map(f => `<li>${esc(f)}</li>`).join('')}</ul><h2>Technology stack</h2>${chips(p.tech)}
 <h2>Architecture</h2><ol class="flow">${(p.arch || ['Frontend', 'API', 'Database']).map(a => `<li>${esc(a)}</li>`).join('')}</ol>
 <h2>UI previews</h2><div class="sg">${[p.mock, nx.mock, (p.mock % 5) + 1].map(m => `<div class="card pv">${mock(m)}</div>`).join('')}</div>
 <h2>Outcome</h2><p class="lead">${esc(p.outcome || '')}</p><div class="cta"><a class="btn pri" href="#" data-demo>Live Demo ↗</a><a class="btn" href="contact.html">Discuss a similar project →</a></div>
 <a class="card nx" href="project-detail.html?project=${encodeURIComponent(nx.slug)}"><small>Next project</small><h3>${esc(nx.title)} →</h3></a></div>`;
}

function initStore() {
 const st = { c: 'all', q: '', s: 'new' };
 makePills($('#pills'), [['all', 'All'], ...Object.entries(CAT)], k => { st.c = k; draw(); });
 on($('#q'), 'input', e => { st.q = e.target.value.toLowerCase(); draw(); });
 on($('#so'), 'change', e => { st.s = e.target.value; draw(); });
 function draw() {
  const l = products.filter(p => (st.c === 'all' || p.category === st.c) && [p.title, p.description, (p.technologies || []).join(' ')].join(' ').toLowerCase().includes(st.q));
  l.sort(st.s === 'lo' ? (a, b) => a.price - b.price : st.s === 'hi' ? (a, b) => b.price - a.price : (a, b) => b.id - a.id);
  out('#sg').innerHTML = l.map(prodCard).join(''); out('#empty').hidden = !!l.length;
 }
 draw();
}

function initProduct() {
 const p = products.find(x => x.id == qs.get('id')) || products[0];
 if (!p) { out('#pr').innerHTML = '<p class="lead">Product not found.</p>'; return; }
 const rel = [...products.filter(x => x.id !== p.id && x.category === p.category), ...products.filter(x => x.id !== p.id && x.category !== p.category)].slice(0, 3);
 const list = a => `<ul class="ul">${a.map(x => `<li>${esc(x)}</li>`).join('')}</ul>`;
 document.title = `${p.title} — Store | Kamol Aliboyev`;
 out('#pr').innerHTML = `<a class="back" href="store.html">← Back to store</a><div class="dh"><div><p class="lab">${esc(CAT[p.category] || p.category)}</p><h1>${esc(p.title)}</h1><div class="big">${vis(p)}</div>
 <h2>Description</h2><p class="lead">${esc(p.description)}</p>${p.longDescription ? `<div class="lead">${sanitizeHtml(p.longDescription)}</div>` : ''}<h2>Features</h2>${list(p.features || [])}<h2>Technologies</h2>${chips(p.technologies)}
 <h2>What's included</h2>${list(['Full source code', 'Documentation', 'Setup instructions'])}<h2>Requirements</h2>${list(['A modern browser', 'Basic knowledge of ' + (p.technologies[0] || 'web technologies'), 'Hosting of your choice'])}
 <h2>FAQ</h2><details><summary>Can I customize it?</summary><p>Yes. You receive the full source code and can change anything.</p></details><details><summary>How do I get support?</summary><p>Use the contact page and describe your question.</p></details><details><summary>Is there a live demo?</summary><p>Use the Live Demo button once a demo link has been added.</p></details></div>
 <aside class="card buy"><div class="price">$${p.price}</div><button class="btn pri" data-buy>Buy Now</button><a class="btn" href="#" data-demo>Live Demo ↗</a><a class="btn" href="#" data-demo>Source Code</a><a class="btn" href="contact.html">Support</a></aside></div>
 <h2 style="margin-top:70px">Related products</h2><div class="sg">${rel.map(prodCard).join('')}</div>`;
}

function initBlog() {
 const st = { c: 'all', q: '' };
 makePills($('#pills'), [['all', 'All'], ...['Development', 'AI', 'SaaS', 'Automation', 'Business', 'Tutorials'].map(c => [c, c])], k => { st.c = k; draw(); });
 on($('#q'), 'input', e => { st.q = e.target.value.toLowerCase(); draw(); });
 function draw() {
  const l = posts.filter(p => (st.c === 'all' || p.category === st.c) && [p.title, p.excerpt].join(' ').toLowerCase().includes(st.q));
  out('#bl').innerHTML = l.map(postCard).join(''); out('#empty').hidden = !!l.length;
 }
 draw();
}

function initArticle() {
 const p = posts.find(x => x.id == qs.get('id')) || posts[0];
 if (!p) { out('#ar').innerHTML = '<p class="lead">Article not found.</p>'; return; }
 const rel = posts.filter(x => x.id !== p.id).slice(0, 2);
 document.title = `${p.title} — Blog | Kamol Aliboyev`;
 out('#ar').innerHTML = `<div class="art"><a class="back" href="blog.html">← Back to Blog</a><div class="meta"><span>${esc(p.category)}</span><span>${esc(p.date)}</span><span>${esc(p.readingTime)} read</span></div><h1>${esc(p.title)}</h1><p class="lead">${esc(p.excerpt)}</p>${safeUrl(p.cover, true) ? `<img class="cover" src="${esc(safeUrl(p.cover, true))}" alt="">` : ''}${sanitizeHtml(p.body || '')}<div class="ph">Diagram placeholder</div></div>
 <h2 style="margin-top:70px">Related articles</h2><div class="bl">${rel.map(postCard).join('')}</div>`;
}

function initContact() {
 const f = $('#fm'); if (!f) return;
 const err = document.createElement('p');
 err.className = 'err'; err.setAttribute('role', 'alert'); f.insertBefore(err, $('button[type=submit]', f));
 f.addEventListener('submit', async e => {
  e.preventDefault();
  if (!f.checkValidity()) { f.reportValidity(); return; }
  const sel = $$('select', f), b = $('button[type=submit]', f); b.disabled = true; err.textContent = '';
  const r = await API.post('/contact', { name: val('#nm'), email: (($('input[type=email]', f) || {}).value || ''), subject: `Project type: ${sel[0] ? sel[0].value : ''} / Budget: ${sel[1] ? sel[1].value : ''}`, message: (($('textarea', f) || {}).value || '') }, { noAuth: true });
  if (!r.ok) { err.textContent = errText(r); b.disabled = false; return; }
  out('#who').textContent = val('#nm'); f.hidden = true; out('#ok').hidden = false;
 });
}

/* ===================== STORE / ACCOUNT / INTERACTION ===================== */
function updateBadge() { const b = $('.cb'); if (b) { const n = Cart.getCount(); b.textContent = n; b.hidden = !n; } }
function decorateHeader() {
 const u = Auth.currentUser(), f = document.createElement('span');
 f.className = 'auth';
 f.innerHTML = `<a class="ic cart-ic" href="cart.html" aria-label="Cart">🛒<span class="cb" hidden>0</span></a>` + (u
  ? `<a class="ic" href="profile.html">Profile</a><button class="ic auth2" id="logout">Logout</button>`
  : `<a class="ic" href="login.html">Login</a><a class="ic auth2" href="register.html">Register</a>`);
 $('.tools', header).insertBefore(f, $('#talk'));
 const lo = $('#logout'); if (lo) lo.addEventListener('click', async () => { await Auth.logout(); location.href = 'index.html'; });
 updateBadge();
}
function applySite() {
 const L = site.links || {}, set = (a, u, t) => { if (/^[^@\s:\/]+@[^@\s]+$/.test(String(u || ''))) u = 'mailto:' + u; u = safeUrl(u); if (a && u) { a.href = u; if (t) a.textContent = t; } }, bare = u => String(u || '').replace(/^https?:\/\/(t\.me\/)?/, '');
 if (page === 'index') { const h = $('.hero h1'), l = $('.hero .lead'); if (h && site.heroTitle) h.innerHTML = esc(site.heroTitle).replace(/\[\[(.+?)\]\]/, '<em>$1</em>'); if (l && site.heroLead) l.textContent = site.heroLead; }
 const f = $$('footer [aria-label=Social] a'); set(f[0], L.github); set(f[1], L.linkedin); set(f[2], L.telegram);
 if (page === 'contact') { const c = $$('.ct a'); set(c[0], L.telegram, '@' + bare(L.telegram)); set(c[1], L.email, String(L.email || '').replace('mailto:', '')); set(c[2], L.github, bare(L.github)); set(c[3], L.linkedin, bare(L.linkedin)); }
}
const safe = n => /^[\w-]+\.html(\?[\w=&%.-]*)?$/.test(n || '') ? n : null;
const copy = async t => { try { await navigator.clipboard.writeText(t); } catch (e) { const a = document.createElement('textarea'); a.value = t; document.body.append(a); a.select(); try { document.execCommand('copy'); } catch (_) {} a.remove(); } toast('Link copied'); };
const share = async title => { if (navigator.share) { try { await navigator.share({ title, url: location.href }); return; } catch (e) { if (e.name === 'AbortError') return; } } copy(location.href); };
const vis = o => o.image ? `<div class="mock img" aria-hidden="true"><img src="${esc(o.image)}" alt="" loading="lazy"></div>` : mock(o.mock || 1);
const access = it => { const id = it.id || it.product_id, p = it.hasDownload !== undefined ? it : products.find(x => x.id == id); return p && p.hasDownload ? `<button class="btn sm pri" data-dl data-id="${id}">Download</button>` : '<small>Download will be available after the product is connected.</small>'; };

/* Demo modal: iframe when a demo URL exists, otherwise a "coming soon" state */
function openDemo(o) {
 const old = $('#demo'); if (old) old.remove();
 const prev = document.activeElement, m = document.createElement('div'), t = esc(o.title);
 const du = safeUrl(o.demoUrl);
 m.id = 'demo'; m.className = 'modal'; m.setAttribute('role', 'dialog'); m.setAttribute('aria-modal', 'true'); m.setAttribute('aria-label', t + ' demo');
 const body = o.demoType === 'html' && o.demoHtml ? `<iframe title="${t} demo" sandbox="allow-scripts allow-forms" srcdoc="${String(o.demoHtml).replace(/&/g, '&amp;').replace(/"/g, '&quot;')}"></iframe>`
  : du ? `<iframe title="${t} demo" src="${esc(du)}" loading="lazy"></iframe>`
  : `<div class="soon">${vis(o)}<h3>Demo preview coming soon</h3><p>A live preview of ${t} will appear here once a demo URL is connected.</p></div>`;
 m.innerHTML = `<div class="mbox"><div class="mhead"><b>Demo · ${t}</b><button class="ic" data-x aria-label="Close demo">×</button></div><div class="mbody">${body}</div><div class="mfoot">${du ? `<a class="btn sm" href="${esc(du)}" target="_blank" rel="noopener">Open Full Demo ↗</a>` : '<span></span>'}<button class="btn sm" data-x>Close</button></div></div>`;
 document.body.appendChild(m); requestAnimationFrame(() => m.classList.add('on'));
 const key = e => { if (e.key === 'Escape') close(); else if (e.key === 'Tab') { const f = $$('a[href],button', m), a = f[0], z = f[f.length - 1]; if (e.shiftKey && document.activeElement === a) { e.preventDefault(); z.focus(); } else if (!e.shiftKey && document.activeElement === z) { e.preventDefault(); a.focus(); } } }, close = () => { document.body.style.overflow = ''; m.classList.remove('on'); setTimeout(() => m.remove(), 200); document.removeEventListener('keydown', key); prev && prev.focus && prev.focus(); };
 m.addEventListener('click', e => { if (e.target === m || e.target.closest('[data-x]')) close(); });
 document.addEventListener('keydown', key); document.body.style.overflow = 'hidden'; $('[data-x]', m).focus();
}

/* Delegated click actions: cart, buy, favorites, demos */
function actions(e) {
 const t = e.target.closest('[data-buy],[data-cart],[data-fav],[data-xdemo],[data-demo],[data-soon],[data-dl]'); if (!t) return false;
 e.preventDefault(); const id = t.dataset.id;
 if (t.matches('[data-cart]')) { Cart.add(id); toast('Added to cart'); }
 else if (t.matches('[data-buy]')) { if (!Cart.raw().some(i => i.id == id)) Cart.add(id); location.href = 'checkout.html'; }
 else if (t.matches('[data-dl]')) { API.get('/products/' + id + '/access').then(r => { if (r.ok && r.data && safeUrl(r.data.download_url)) window.open(safeUrl(r.data.download_url), '_blank', 'noopener'); else toast(r.ok ? 'Download will be available after the product is connected.' : r.message); }); }
 else if (t.matches('[data-fav]')) { if (!Auth.isLoggedIn()) { toast('Please login to save favorites.'); return true; } const on = Favs.toggle(id); t.setAttribute('aria-pressed', on); t.textContent = t.classList.contains('fav') ? (on ? '♥' : '♡') : (on ? '♥ Remove from favorites' : '♡ Add to favorites'); toast(on ? 'Added to favorites' : 'Removed from favorites'); }
 else if (t.matches('[data-xdemo]')) { const p = products.find(x => x.id == id); if (p) openDemo(p); }
 else if (t.matches('[data-demo]')) {
  const c = t.closest('.pc,.proj'), l = c && $('a[href*="project="]', c), slug = c ? (l ? new URL(l.href).searchParams.get('project') : null) : qs.get('project'), p = projects.find(x => x.slug === slug);
  if (p) openDemo({ title: p.title, demoUrl: p.demoUrl || p.liveUrl, mock: p.mock, image: p.image }); else toast('Demo link not set yet.');
 } else toast('Link not set yet.');
 return true;
}

/* Likes + comments + share, used by blog articles and project pages (all stored on the server) */
function discussion(el, type, id, info) {
 const base = (type === 'blog' ? '/posts/' : '/projects/') + id, nx = encodeURIComponent(location.pathname.split('/').pop() + location.search);
 let comments = [];
 el.className = 'disc';
 const draw = () => {
  const u = Auth.currentUser(), top = comments.filter(c => !c.parent_id);
  const row = (c, r) => `<div class="cm${r ? ' rp' : ''}"><span class="av">${esc(ini(c.name))}</span><div><b>${esc(c.name)}</b> <small>${fmt(c.created_at)}</small><p>${esc(c.content)}</p><div class="ca"><button data-cl="${c.id}" class="${c.liked ? 'on' : ''}" aria-label="Like comment">${c.liked ? '♥' : '♡'} ${esc(c.like_count)}</button>${r ? '' : `<button data-rp="${c.id}">Reply</button>`}${u && u.id === c.user_id ? `<button data-del="${c.id}">Delete</button>` : ''}</div>${r ? '' : `<div class="rpf" data-for="${c.id}"></div>`}</div></div>`;
  el.innerHTML = `<h2>${type === 'blog' ? 'Discussion' : 'Project discussion'}</h2><div class="ia"><button class="btn sm" data-like aria-pressed="${!!info.liked}">${info.liked ? '♥ Liked' : '♡ Like'} · ${esc(info.likes || 0)}</button><button class="btn sm" data-share>Share ↗</button><small>${comments.length} comment${comments.length === 1 ? '' : 's'}</small></div>`
   + (u ? `<form class="fm" data-cf><label class="sr" for="ct-${type}">Comment</label><textarea id="ct-${type}" rows="3" placeholder="Join the discussion…" required></textarea><p class="err" data-err role="alert"></p><button class="btn pri sm" type="submit">Post Comment</button></form>`
    : `<div class="card cform"><p>Please login to join the discussion.</p><div class="cta"><a class="btn pri" href="login.html?next=${nx}">Login</a><a class="btn" href="register.html?next=${nx}">Register</a></div></div>`)
   + `<div>${top.map(c => row(c) + comments.filter(r => r.parent_id === c.id).map(r => row(r, 1)).join('')).join('') || '<p class="lead">No comments yet.</p>'}</div>`;
 };
 const load = async () => { const r = await API.get(base + '/comments'); comments = r.ok && Array.isArray(r.data) ? r.data : []; draw(); };
 el.addEventListener('click', async e => {
  const b = e.target.closest('button'); if (!b) return;
  if ('share' in b.dataset) { share(document.title); return; }
  if ('like' in b.dataset || b.dataset.cl) {
   if (!Auth.isLoggedIn()) { toast('Please login to like.'); return; }
   const c = b.dataset.cl ? comments.find(x => x.id == b.dataset.cl) : null, liked = c ? c.liked : info.liked;
   const r = await API.req(liked ? 'DELETE' : 'POST', (c ? '/comments/' + c.id : base) + '/like', liked ? undefined : {});
   if (!r.ok) { toast(r.message); return; }
   if (c) { c.liked = r.data.liked; c.like_count = r.data.like_count; } else { info.liked = r.data.liked; info.likes = r.data.like_count; }
   draw();
  } else if (b.dataset.del) { const r = await API.del('/comments/' + b.dataset.del); r.ok ? load() : toast(r.message); }
  else if (b.dataset.rp) {
   if (!Auth.isLoggedIn()) { toast('Please login to reply.'); return; }
   const box = $(`.rpf[data-for="${b.dataset.rp}"]`, el); box.innerHTML = `<form class="fm" data-rf="${b.dataset.rp}"><label class="sr" for="r-${b.dataset.rp}">Reply</label><textarea id="r-${b.dataset.rp}" rows="2" required></textarea><p class="err" data-err role="alert"></p><button class="btn sm pri" type="submit">Reply</button></form>`; $('textarea', box).focus();
  }
 });
 el.addEventListener('submit', async e => {
  e.preventDefault(); const f = e.target, ta = $('textarea', f), btn = $('button', f);
  btn.disabled = true;
  const r = 'cf' in f.dataset ? await API.post(base + '/comments', { content: ta.value }) : await API.post('/comments/' + f.dataset.rf + '/reply', { content: ta.value });
  if (!r.ok) { $('[data-err]', f).textContent = errText(r); btn.disabled = false; return; }
  load();
 });
 draw(); load();
}

/* ----- Cart / checkout / payment pages ----- */
function initCart() {
 const el = out('#cart');
 const draw = () => {
  const it = Cart.getItems();
  if (!it.length) { el.innerHTML = '<div class="card cform c"><h3>Your cart is empty</h3><p class="lead">Browse the store and add a product.</p><a class="btn pri" href="store.html">Continue shopping →</a></div>'; return; }
  el.innerHTML = `<div class="two cw"><div class="cl">${it.map(i => `<div class="card ci"><a class="pv" href="product.html?id=${i.id}" aria-label="${esc(i.product.title)}">${vis(i.product)}</a><div><small>${CAT[i.product.category] || ''}</small><h3>${esc(i.product.title)}</h3><b>$${i.product.price}</b></div><div class="qty"><button class="ic" data-dec="${i.id}" aria-label="Decrease quantity">−</button><span aria-live="polite">${i.qty}</span><button class="ic" data-inc="${i.id}" aria-label="Increase quantity">+</button></div><div class="sub">$${i.product.price * i.qty}</div><button class="ic" data-rm="${i.id}" aria-label="Remove ${esc(i.product.title)}">✕</button></div>`).join('')}</div>
  <aside class="card cform sum"><h3>Order summary</h3>${it.map(i => `<div class="ln"><span>${esc(i.product.title)} × ${i.qty}</span><b>$${i.product.price * i.qty}</b></div>`).join('')}<div class="ln tot"><span>Total</span><b>$${Cart.getTotal()}</b></div><a class="btn pri" href="checkout.html">Proceed to checkout →</a><a class="btn" href="store.html">Continue shopping</a><button class="btn" data-clear>Clear cart</button></aside></div>`;
 };
 el.addEventListener('click', e => {
  const t = e.target.closest('button'); if (!t) return;
  if (t.dataset.inc) Cart.increase(t.dataset.inc); else if (t.dataset.dec) Cart.decrease(t.dataset.dec); else if (t.dataset.rm) Cart.remove(t.dataset.rm); else if ('clear' in t.dataset) Cart.clear(); else return;
  draw();
 });
 draw();
}
const orderSum = (items, total, title) => `<aside class="card cform sum"><h3>${esc(title)}</h3>${items.map(i => `<div class="ln"><span>${esc(i.title || i.product.title)} × ${i.qty}</span><b>$${((i.price != null ? i.price : i.product.price) * i.qty).toFixed(2).replace(/\.00$/, '')}</b></div>`).join('')}<div class="ln tot"><span>Total</span><b>$${Number(total).toFixed(2).replace(/\.00$/, '')}</b></div></aside>`;
const sleep = ms => new Promise(r => setTimeout(r, ms));

function initCheckout() {
 const el = out('#co'), it = Cart.getItems(), u = Auth.currentUser();
 if (!it.length) { el.innerHTML = '<div class="card cform c"><h3>Your cart is empty</h3><a class="btn pri" href="store.html">Continue shopping →</a></div>'; return; }
 if (!u) { el.innerHTML = '<div class="card cform c"><h3>Please log in to continue</h3><p class="lead">You need an account to place an order.</p><div class="cta" style="justify-content:center"><a class="btn pri" href="login.html?next=checkout.html">Login</a><a class="btn" href="register.html?next=checkout.html">Register</a></div></div>'; return; }
 el.innerHTML = `<div class="two cw"><form class="card cform fm" id="cof" novalidate><h3>Customer information</h3><label>Full Name<input id="fn" value="${esc(u.name)}" autocomplete="name" required></label><label>Email<input id="fe" type="email" value="${esc(u.email)}" autocomplete="email" required></label><div class="r2"><label>Phone<input id="fp" type="tel" autocomplete="tel" required></label><label>Country<input id="fc" autocomplete="country-name" required></label></div>
 <label>Promo code (optional)<span class="opt"><input id="pc" style="flex:1" placeholder="Promo code"><button class="btn" type="button" id="pa">Apply</button></span></label><p class="err" id="err" role="alert"></p><button class="btn pri" id="cb" type="submit">Continue to payment →</button></form>${orderSum(it, Cart.getTotal(), 'Order summary')}</div>`;
 on($('#pa'), 'click', () => toast('The promo code is checked when you continue.'));
 on($('#cof'), 'submit', async e => {
  e.preventDefault(); const v = id => $('#' + id).value.trim(), b = $('#cb');
  if (!v('fn') || !/^\S+@\S+\.\S+$/.test(v('fe')) || !v('fp') || !v('fc')) { out('#err').textContent = 'Please complete all fields with a valid email.'; return; }
  // The same idempotency key is reused for the same cart so a double click or retry cannot create two orders.
  const sig = JSON.stringify(it.map(i => [i.id, i.qty])); let idem = null; try { idem = JSON.parse(sessionStorage.getItem('kamol_idem') || 'null'); } catch (_) {}
  if (!idem || idem.sig !== sig) { idem = { sig, key: (crypto.randomUUID ? crypto.randomUUID() : String(Date.now()) + Math.random()) }; try { sessionStorage.setItem('kamol_idem', JSON.stringify(idem)); } catch (_) {} }
  b.disabled = true; out('#err').textContent = '';
  const r = await API.post('/orders', { customer: { name: v('fn'), email: v('fe'), phone: v('fp'), country: v('fc') }, items: it.map(i => ({ product_id: i.id, quantity: i.qty })), coupon: v('pc') }, { headers: { 'Idempotency-Key': idem.key } });
  if (!r.ok) { out('#err').textContent = errText(r); b.disabled = false; return; }
  location.href = 'payment.html?order=' + r.data.id;
 });
}

async function initPayment() {
 const el = out('#pay'), notice = t => { el.innerHTML = `<div class="card cform c"><h3>${t}</h3><a class="btn pri" href="store.html">Back to store</a></div>`; };
 if (!Auth.currentUser()) { notice('Please log in to continue'); return; }
 const r = await API.get('/orders/' + encodeURIComponent(qs.get('order') || ''));
 if (!r.ok) { notice('Nothing to pay for'); return; }
 const o = normOrder(r.data);
 const done = ord => { el.innerHTML = `<div class="card cform c"><h2>Payment successful</h2><p class="lead" style="margin-inline:auto">Order <b>#${esc(ord.number)}</b>${boot.offline ? '' : ''}</p><div style="text-align:left">${orderSum(ord.items, ord.total, 'Order summary').replace('sum"', 'sum" style="position:static"')}<h3 style="margin:22px 0 8px">Access your products</h3>${ord.items.map(i => `<div class="ln"><span>${esc(i.title)}</span>${access(i)}</div>`).join('')}</div><div class="cta" style="justify-content:center"><a class="btn pri" href="profile.html">View order history</a><a class="btn" href="store.html">Continue shopping</a></div></div>`; };
 if (o.status === 'CANCELLED' || o.status === 'REFUNDED') { notice('This order is ' + o.status.toLowerCase()); return; }
 if (o.status !== 'PENDING') { done(o); return; }
 el.innerHTML = `<div class="two cw"><form class="card cform fm" id="pf" novalidate><h3>Payment method</h3><div class="opt"><label><input type="radio" name="m" value="card" checked> Bank Card</label><label><input type="radio" name="m" value="online"> Online Payment</label><label><input type="radio" name="m" value="other"> Other</label></div>
 <div class="fm" id="cf"><label>Cardholder Name<input id="cn" autocomplete="cc-name"></label><label>Card Number<input id="cc" inputmode="numeric" autocomplete="cc-number" placeholder="•••• •••• •••• ••••"></label><div class="r2"><label>Expiry Date<input id="ce" placeholder="MM/YY" autocomplete="cc-exp"></label><label>CVV<input id="cv" type="password" inputmode="numeric" maxlength="4" autocomplete="cc-csc"></label></div></div>
 <p class="lead" id="np" hidden>This method is a placeholder. The test payment completes without extra details.</p><p class="err" id="err" role="alert"></p><button class="btn pri" id="pb" type="submit">Pay now</button><small>Test mode: no real payment is processed. Card details are checked in your browser only and are never sent to or stored by the server.</small></form>${orderSum(o.items, o.total, 'Order #' + o.number)}</div>`;
 const f = $('#pf'); f.addEventListener('change', () => { const card = $('input[name=m]:checked', f).value === 'card'; out('#cf').hidden = !card; out('#np').hidden = card; });
 f.addEventListener('submit', async e => {
  e.preventDefault(); const m = $('input[name=m]:checked', f).value, v = id => $('#' + id).value.trim(), fail = t => { out('#err').textContent = t; b.disabled = false; b.textContent = 'Pay now'; }, b = $('#pb');
  if (m === 'card' && (!v('cn') || !/^\d{13,19}$/.test(v('cc').replace(/\s/g, '')) || !/^(0[1-9]|1[0-2])\/\d{2}$/.test(v('ce')) || !/^\d{3,4}$/.test(v('cv')))) { out('#err').textContent = 'Check the card details: name, 13–19 digit number, MM/YY expiry and CVV.'; return; }
  out('#err').textContent = ''; b.disabled = true; b.textContent = 'Processing…';
  const p = await API.post('/payments', { order_id: o.id });
  if (!p.ok) return fail(p.message);
  if (p.data && safeUrl(p.data.checkout_url)) { location.href = safeUrl(p.data.checkout_url); return; } // real provider: hosted payment page
  if (p.data && p.data.provider === 'test') { const c = await API.post('/payments/test/complete', { payment_id: p.data.payment_id, outcome: 'success' }); if (!c.ok) return fail(c.message); }
  // The server decides: wait until the verified webhook has marked the order as paid.
  for (let i = 0; i < 12; i++) {
   const s = await API.get('/orders/' + o.id);
   if (s.ok && s.data.status !== 'PENDING') { await Cart.sync(); done(normOrder(s.data)); return; }
   if (s.ok && s.data.payment_status === 'FAILED') return fail('The payment was declined. Please try again.');
   await sleep(700);
  }
  fail('Payment is still processing. Check your order history in a minute.');
 });
}

/* ----- Auth pages ----- */
function initLogin() {
 if (Auth.isLoggedIn()) { location.replace('profile.html'); return; }
 const nx = qs.get('next'); if (nx) out('#alt').href = 'register.html?next=' + encodeURIComponent(nx);
 const reset = qs.get('reset');
 on($('#fg'), 'click', async e => {
  e.preventDefault();
  if (reset) { const pw = prompt('Enter a new password (at least 8 characters):'); if (!pw) return; const r = await API.post('/auth/reset-password', { token: reset, password: pw }, { noAuth: true }); toast(r.ok ? 'Password updated. Please log in.' : errText(r)); return; }
  const email = val('#em').trim(); if (!email) { toast('Enter your email first, then press "Forgot password?".'); return; }
  const r = await API.post('/auth/forgot-password', { email }, { noAuth: true }); toast(r.ok ? 'If an account exists, a reset link has been sent.' : r.message);
 });
 if (reset) out('#fg').textContent = 'Set a new password with your reset link';
 on($('#lf'), 'submit', async e => {
  e.preventDefault(); const b = $('#lf button[type=submit]'); b.disabled = true;
  const r = await Auth.login(val('#em'), val('#pw'), !!($('#rm') || {}).checked);
  if (!r.ok) { out('#err').textContent = r.error; b.disabled = false; return; }
  location.href = safe(nx) || 'profile.html';
 });
}
function initRegister() {
 if (Auth.isLoggedIn()) { location.replace('profile.html'); return; }
 const nx = qs.get('next'); if (nx) out('#alt').href = 'login.html?next=' + encodeURIComponent(nx);
 on($('#rf'), 'submit', async e => {
  e.preventDefault(); const b = $('#rf button[type=submit]'); b.disabled = true;
  const r = await Auth.register({ name: val('#nm'), email: val('#em'), password: val('#pw'), confirm: val('#pw2') });
  if (!r.ok) { out('#err').textContent = r.error; b.disabled = false; return; }
  location.href = safe(nx) || 'profile.html';
 });
}
async function initProfile() {
 const u = Auth.currentUser(); if (!u) { location.replace('login.html?next=profile.html'); return; }
 const [o, pu, fv] = await Promise.all([API.get('/me/orders?limit=50'), API.get('/me/purchases'), API.get('/me/favorites')]);
 const arr = x => x.ok && Array.isArray(x.data) ? x.data : [];
 const orders = arr(o).map(normOrder), purchases = arr(pu), favs = arr(fv).map(mapProduct);
 out('#pf').innerHTML = `<div class="pfh"><span class="av lg">${esc(ini(u.name))}</span><div><h1 style="margin:0">${esc(u.name)}</h1><p class="lead" style="margin:0">${esc(u.email)} · Member since ${fmt(u.created)}</p></div><button class="btn" id="lo">Logout</button></div>
 <h2>Order history</h2>${orders.map(x => `<div class="card ord"><div><b>#${esc(x.number)}</b><br><small>${fmt(x.date)}</small></div><div>${x.items.map(i => `${esc(i.title)} × ${i.qty}`).join('<br>')}</div><b>$${x.total}</b><span class="st st-${titleCase(x.status)}">${titleCase(x.status)}</span></div>`).join('') || '<p class="lead">No orders yet.</p>'}
 <h2 style="margin-top:40px">My purchases</h2>${purchases.map(x => `<div class="card ord pur"><b>${esc(x.product.title)}</b><small>Since ${fmt(x.granted_at)}</small><span>${access({ id: x.product.id, hasDownload: x.product.has_download })}</span></div>`).join('') || '<p class="lead">Purchased products appear here after payment.</p>'}
 <h2 style="margin-top:40px">Favorites</h2>${favs.length ? `<div class="sg">${favs.map(prodCard).join('')}</div>` : '<p class="lead">No favorites yet. Use the ♡ on any store product.</p>'}`;
 on($('#lo'), 'click', async () => { await Auth.logout(); location.href = 'index.html'; });
}

/* ----- Wrappers: extend existing pages without rewriting them ----- */
const _ip = initProduct; initProduct = function () {
 _ip(); const p = products.find(x => x.id == qs.get('id')) || products[0]; if (!p) return;
 out('.buy').innerHTML = `<div class="price">$${p.price}${p.oldPrice ? ` <s class="old">$${p.oldPrice}</s>` : ''}</div><button class="btn pri" data-buy data-id="${p.id}">Buy Now</button><button class="btn" data-cart data-id="${p.id}">Add to Cart</button><button class="btn" data-xdemo data-id="${p.id}">Demo</button><button class="btn" data-fav data-id="${p.id}" aria-pressed="${Favs.has(p.id)}">${Favs.has(p.id) ? '♥ Remove from favorites' : '♡ Add to favorites'}</button><a class="btn" href="#" data-soon>Source Code</a><a class="btn" href="contact.html">Support</a>`;
};
const _id = initDetail; initDetail = function () {
 _id(); const p = projects.find(x => x.slug === qs.get('project')) || projects[0]; if (!p || !$('.cs')) return; const s = document.createElement('section'); $('.cs').append(s); discussion(s, 'project', p.id, { likes: p.likes, liked: p.liked });
 if (safeUrl(p.githubUrl) && $('.cs .cta')) $('.cs .cta').insertAdjacentHTML('beforeend', `<a class="btn" href="${esc(safeUrl(p.githubUrl))}" target="_blank" rel="noopener">GitHub ↗</a>`);
};
const _ia = initArticle; initArticle = function () {
 _ia(); const p = posts.find(x => x.id == qs.get('id')) || posts[0]; if (!p || !$('.art')) return; const s = document.createElement('section'); $('.art').append(s); discussion(s, 'blog', p.id, { likes: p.likes, liked: p.liked });
};

addEventListener('pageshow', updateBadge); addEventListener('storage', updateBadge);
const pageInit = { cart: initCart, checkout: initCheckout, payment: initPayment, login: initLogin, register: initRegister, profile: initProfile, index: initHome, skills: initSkills, projects: initProjects, 'project-detail': initDetail, store: initStore, product: initProduct, blog: initBlog, article: initArticle, contact: initContact }[page];
try { const res = pageInit && pageInit(); if (res && typeof res.catch === 'function') res.catch(e => console.error('Page init failed:', e)); } catch (e) { console.error('Page init failed:', e); }

/* ===================== GLOBAL BEHAVIOUR ===================== */
// Placeholder actions
document.addEventListener('click', e => {
 const b = e.target.closest('[data-buy]'), d = e.target.closest('[data-demo]');
 if (actions(e)) return;
 
 // Page transition: fade out, navigate
 const a = e.target.closest('a[href]');
 if (!a || e.defaultPrevented || e.metaKey || e.ctrlKey || e.shiftKey || a.target || /^(#|mailto:|https?:)/.test(a.getAttribute('href'))) return;
 if (a.pathname === location.pathname && a.search === location.search) return;
 e.preventDefault(); setMenu(false);
 if (reduce) { location.href = a.href; return; }
 document.body.classList.add('leaving'); setTimeout(() => { location.href = a.href; }, 170);
});
addEventListener('pageshow', () => document.body.classList.remove('leaving'));

// Scroll reveal
const io = new IntersectionObserver(es => es.forEach(e => { if (e.isIntersecting) { e.target.classList.add('in'); io.unobserve(e.target); } }), { threshold: .12 });
$$('.reveal').forEach(el => io.observe(el));

// Cursor glow (desktop) and card glow
if (fine && !reduce && $('.glow')) { const g = $('.glow'); addEventListener('pointermove', e => { g.style.transform = `translate(${e.clientX - 180}px,${e.clientY - 180}px)`; }, { passive: true }); }
document.addEventListener('pointermove', e => { const c = e.target.closest && e.target.closest('.card'); if (c) { const r = c.getBoundingClientRect(); c.style.setProperty('--mx', e.clientX - r.left + 'px'); c.style.setProperty('--my', e.clientY - r.top + 'px'); } });
})().catch(e => console.error('app.js failed:', e));
