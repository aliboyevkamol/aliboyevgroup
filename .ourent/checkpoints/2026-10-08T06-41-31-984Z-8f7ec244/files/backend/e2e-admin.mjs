// Admin panel + API end-to-end test.
const BASE = 'http://localhost:8080', API = BASE + '/api/v1';
const results = []; let cookies = {};
const ok = (n, c, extra = '') => { results.push([c ? 'PASS' : 'FAIL', n, extra]); console.log(`${c ? 'PASS' : 'FAIL'} | ${n} ${extra}`); };
function store(res) { const raw = res.headers.getSetCookie ? res.headers.getSetCookie() : []; for (const c of raw) { const [kv] = c.split(';'); const i = kv.indexOf('='); cookies[kv.slice(0, i)] = kv.slice(i + 1); } }
async function req(method, path, body, token, extraHeaders = {}) {
  const h = { ...extraHeaders }; if (body !== undefined) h['Content-Type'] = 'application/json'; if (token) h.Authorization = 'Bearer ' + token;
  const ck = Object.entries(cookies).map(([k, v]) => `${k}=${v}`).join('; '); if (ck) h.Cookie = ck;
  const res = await fetch(API + path, { method, headers: h, body: body === undefined ? undefined : JSON.stringify(body) }); store(res);
  let json = null; try { json = await res.json(); } catch (e) {}
  return { status: res.status, body: json };
}
(async () => {
  const login = await req('POST', '/auth/login', { email: 'admin@local.dev', password: 'admin_password_12345' });
  const token = login.body?.data?.access_token;
  ok('admin login', login.status === 200 && !!token, `-> ${login.status} role=${login.body?.data?.user?.role}`);

  const dash = await req('GET', '/admin/dashboard', undefined, token);
  ok('admin dashboard', dash.status === 200, `-> ${dash.status} ${JSON.stringify(dash.body?.data).slice(0, 220)}`);

  for (const [n, p] of [['products', '/admin/products'], ['projects', '/admin/projects'], ['posts', '/admin/posts'], ['orders', '/admin/orders'], ['users', '/admin/users'], ['comments', '/admin/comments'], ['portfolio', '/admin/portfolio'], ['media', '/admin/media'], ['contact-messages', '/admin/contact-messages']]) {
    const r = await req('GET', p, undefined, token);
    const c = Array.isArray(r.body?.data) ? r.body.data.length : 1;
    ok('admin list ' + n, r.status === 200, `-> ${r.status}, ${c} rows`);
  }

  const bad = await req('POST', '/admin/products', { title: '', category: 'nope', price: -5 }, token);
  ok('admin product validation', bad.status === 422, `-> ${bad.status}`);

  const created = await req('POST', '/admin/products', { title: 'E2E Test Product', category: 'bots', price: 19.99, description: 'created by the e2e suite', technologies: ['Go'], features: ['A'], download_url: 'private/e2e.zip', published: false }, token);
  const pid = created.body?.data?.id;
  ok('admin create product', (created.status === 201 || created.status === 200) && !!pid, `-> ${created.status} id=${pid}`);

  const pubList = await req('GET', '/products?limit=100');
  ok('unpublished product hidden from public API', !pubList.body.data.some(p => p.id === pid), '');
  ok('public API still hides download_url', pubList.body.data.every(p => p.download_url === undefined), '');

  const patched = await req('PATCH', `/admin/products/${pid}`, { published: true, featured: true, price: 24.5 }, token);
  ok('admin patch product', patched.status === 200, `-> ${patched.status}`);
  const pub2 = await req('GET', '/products?limit=100');
  const nowPub = pub2.body.data.find(p => p.id === pid);
  ok('published product now public', !!nowPub && Number(nowPub.price) === 24.5, `price=${nowPub?.price}`);
  ok('download_url still hidden after publish', nowPub?.download_url === undefined, `keys has download_url: ${nowPub && 'download_url' in nowPub}`);

  const post = await req('POST', '/admin/posts', { title: 'E2E Generated Post', category: 'AI', content: 'word '.repeat(450), published: true, tags: ['go'] }, token);
  ok('admin create post', post.status === 201 || post.status === 200, `-> ${post.status}`);
  const proj = await req('POST', '/admin/projects', { title: 'E2E Project', category: 'SaaS', features: ['f1'], technologies: ['Go'], published: true }, token);
  ok('admin create project', proj.status === 201 || proj.status === 200, `-> ${proj.status}`);

  const port = await req('PUT', '/admin/portfolio', { hero_title: 'Hello [[E2E]]', skills: [{ category: 'backend', name: 'Go', level: 90 }] }, token);
  ok('admin update portfolio', port.status === 200, `-> ${port.status}`);
  const pf = await req('GET', '/portfolio');
  ok('public portfolio reflects update', pf.body.data.hero_title === 'Hello [[E2E]]' && pf.body.data.skills.length === 1, `title=${pf.body.data.hero_title}`);

  if (pid) {
    const del = await req('DELETE', `/admin/products/${pid}`, undefined, token);
    const del2 = await req('DELETE', `/admin/products/${pid}`, undefined, token);
    ok('admin delete product', del.status === 204 && del2.status === 404, `-> ${del.status}/${del2.status}`);
  }

  const anon = await req('GET', '/admin/dashboard');
  const anon2 = await req('GET', '/admin/dashboard', undefined, 'not-a-real-token');
  ok('admin API protected (anon + bad token)', anon.status === 401 && anon2.status === 401, `-> ${anon.status}/${anon2.status}`);

  // Static admin assets
  for (const p of ['/admin/', '/admin/app.js', '/admin/style.css']) { const r = await fetch(BASE + p); ok('static ' + p, r.status === 200, `-> ${r.status}`); }

  const passed = results.filter(r => r[0] === 'PASS').length, failed = results.filter(r => r[0] === 'FAIL').length;
  console.log(`\n==== ADMIN E2E: ${passed} passed, ${failed} failed, ${results.length} total ====`);
  if (failed) console.log(results.filter(r => r[0] === 'FAIL').map(r => ' - ' + r[1] + ' ' + r[2]).join('\n'));
})();
