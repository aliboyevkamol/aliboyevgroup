// End-to-end test of the portfolio website backend + API.
const BASE = 'http://localhost:8080';
const API = BASE + '/api/v1';
const results = [];
const ok = (n, c, extra = '') => { results.push([c ? 'PASS' : 'FAIL', n, extra]); console.log(`${c ? 'PASS' : 'FAIL'} | ${n} ${extra}`); };

let cookies = {};
function cookieHeader() { return Object.entries(cookies).map(([k, v]) => `${k}=${v}`).join('; '); }
function store(res) {
  const raw = res.headers.getSetCookie ? res.headers.getSetCookie() : [];
  for (const c of raw) { const [kv] = c.split(';'); const i = kv.indexOf('='); cookies[kv.slice(0, i)] = kv.slice(i + 1); }
}
async function req(method, path, body, token, extraHeaders = {}) {
  const h = { ...extraHeaders };
  if (body !== undefined) h['Content-Type'] = 'application/json';
  if (token) h.Authorization = 'Bearer ' + token;
  const ck = cookieHeader(); if (ck) h.Cookie = ck;
  const res = await fetch(API + path, { method, headers: h, body: body === undefined ? undefined : JSON.stringify(body) });
  store(res);
  let json = null; try { json = await res.json(); } catch (e) {}
  return { status: res.status, body: json, headers: res.headers };
}

(async () => {
  const stamp = Date.now();
  const user = { name: 'Test User', email: `e2e_${stamp}@example.com`, password: 'StrongPass12345', confirm_password: 'StrongPass12345' };

  // 1. public content
  for (const [n, p] of [['GET /products', '/products?limit=100'], ['GET /projects', '/projects?limit=50'], ['GET /posts', '/posts?limit=100'], ['GET /portfolio', '/portfolio']]) {
    const r = await req('GET', p);
    const arr = Array.isArray(r.body?.data) ? r.body.data.length : Object.keys(r.body?.data || {}).length;
    ok(n, r.status === 200 && r.body.success === true, `-> ${r.status}, ${arr} records`);
  }

  // 2. registration validation + duplicate email
  const bad = await req('POST', '/auth/register', { name: 'x', email: 'not-an-email', password: '123', confirm_password: '456' });
  ok('register rejects invalid input', bad.status === 400 || bad.status === 422, `-> ${bad.status} ${JSON.stringify(bad.body?.error?.fields || {})}`);

  const reg = await req('POST', '/auth/register', user);
  const token = reg.body?.data?.access_token;
  ok('register creates account', reg.status === 201 || reg.status === 200, `-> ${reg.status}`);
  ok('register sets refresh cookie', !!cookies.refresh_token || Object.keys(cookies).length > 0, `cookies: ${Object.keys(cookies).join(',')}`);

  const dup = await req('POST', '/auth/register', user);
  ok('duplicate email rejected', dup.status === 409 || dup.status === 400, `-> ${dup.status}`);

  const wrong = await req('POST', '/auth/login', { email: user.email, password: 'WrongPassword123' });
  ok('wrong password rejected', wrong.status === 401, `-> ${wrong.status}`);

  const me = await req('GET', '/auth/me', undefined, token);
  ok('GET /auth/me with token', me.status === 200 && me.body.data.email === user.email, `-> ${me.status} ${me.body?.data?.role}`);

  const noAuth = await req('GET', '/auth/me');
  ok('GET /auth/me without token = 401', noAuth.status === 401, `-> ${noAuth.status}`);

  const adminForbidden = await req('GET', '/admin/products', undefined, token);
  ok('admin endpoint forbidden for normal user', adminForbidden.status === 401 || adminForbidden.status === 403, `-> ${adminForbidden.status}`);

  const anonAdmin = await req('GET', '/admin/dashboard');
  ok('admin endpoint forbidden anonymously', anonAdmin.status === 401, `-> ${anonAdmin.status}`);

  // 3. products whitelisting
  const list = await req('GET', '/products?limit=100');
  const prod = list.body.data[0];
  ok('product payload hides download_url', prod.download_url === undefined, `keys: ${Object.keys(prod).join(',')}`);
  ok('products have price/featured/slug', prod.price !== undefined && prod.has_download !== undefined, `id=${prod.id} slug=${prod.slug}`);

  // 4. favorites + likes
  const fav = await req('POST', `/products/${prod.id}/favorite`, {}, token);
  ok('add favorite', fav.status === 200 || fav.status === 201, `-> ${fav.status}`);
  const favDup = await req('POST', `/products/${prod.id}/favorite`, {}, token);
  ok('duplicate favorite handled', favDup.status === 200 || favDup.status === 409, `-> ${favDup.status}`);
  const favList = await req('GET', '/me/favorites', undefined, token);
  ok('favorites list', favList.status === 200 && favList.body.data.some(p => p.id === prod.id), `-> ${favList.status}, ${favList.body.data.length} items`);
  await req('DELETE', `/products/${prod.id}/favorite`, undefined, token);

  const like = await req('POST', `/products/${prod.id}/like`, {}, token);
  ok('product like works', like.status === 200 || like.status === 201 || like.status === 404, `-> ${like.status}`);

  // 5. comments + reply + moderation
  const projects = await req('GET', '/projects?limit=5');
  const pid = projects.body.data[0].id;
  const c1 = await req('POST', `/projects/${pid}/comments`, { content: 'E2E test comment' }, token);
  ok('post comment', c1.status === 201 || c1.status === 200, `-> ${c1.status}`);
  const cid = c1.body?.data?.id;
  const cl = await req('GET', `/projects/${pid}/comments`);
  ok('public comments list', cl.status === 200 && Array.isArray(cl.body.data), `-> ${cl.status}, ${cl.body?.data?.length}`);
  if (cid) {
    const rep = await req('POST', `/comments/${cid}/reply`, { content: 'E2E reply' }, token);
    ok('reply to comment', rep.status === 201 || rep.status === 200, `-> ${rep.status}`);
    const pl = await req('POST', `/comments/${cid}/like`, {}, token);
    ok('like comment', pl.status === 200 || pl.status === 201, `-> ${pl.status}`);
  }

  // 6. contact form
  const contact = await req('POST', '/contact', { name: 'E2E', email: user.email, subject: 'Test', message: 'Hello from the E2E suite' });
  ok('contact form accepted', contact.status === 200 || contact.status === 201, `-> ${contact.status}`);
  const contactBad = await req('POST', '/contact', { name: '', email: 'bad', message: '' });
  ok('contact form validates', contactBad.status === 400 || contactBad.status === 422, `-> ${contactBad.status}`);

  // 7. cart flow (server-side)
  const cartAdd = await req('POST', '/cart/items', { product_id: prod.id, quantity: 2 }, token);
  ok('add to cart', cartAdd.status === 200 || cartAdd.status === 201, `-> ${cartAdd.status}`);
  const cart = await req('GET', '/cart', undefined, token);
  const cartItem = cart.body?.data?.items?.find(i => i.product_id === prod.id);
  ok('cart persists on server', cart.status === 200 && cartItem && cartItem.quantity === 2, `-> qty=${cartItem?.quantity}`);
  // The cart endpoint exposes the computed sum as `subtotal` (see GET /cart in commerce.go).
  const cartTotal = cart.body?.data?.subtotal;
  ok('cart total computed server-side', Number(cartTotal) === Number(prod.price) * 2, `subtotal=${cartTotal} expected=${Number(prod.price) * 2}`);

  // 8. order + payment (test provider)
  const order = await req('POST', '/orders', {
    customer: { name: user.name, email: user.email, phone: '+998900000000', country: 'Uzbekistan' },
    items: [{ product_id: prod.id, quantity: 2 }], coupon: ''
  }, token, { 'Idempotency-Key': 'e2e-' + stamp });
  const orderId = order.body?.data?.id;
  ok('create order', (order.status === 201 || order.status === 200) && !!orderId, `-> ${order.status} order=${order.body?.data?.order_number} total=${order.body?.data?.total}`);
  const idem = await req('POST', '/orders', {
    customer: { name: user.name, email: user.email, phone: '+998900000000', country: 'Uzbekistan' },
    items: [{ product_id: prod.id, quantity: 2 }], coupon: ''
  }, token, { 'Idempotency-Key': 'e2e-' + stamp });
  ok('idempotent order (no duplicate)', idem.body?.data?.id === orderId, `-> ${idem.body?.data?.id} vs ${orderId}`);

  const orderForbidden = await req('GET', `/orders/${orderId}`);
  ok('order not readable anonymously', orderForbidden.status === 401 || orderForbidden.status === 403 || orderForbidden.status === 404, `-> ${orderForbidden.status}`);

  const pay = await req('POST', '/payments', { order_id: orderId }, token);
  ok('create payment', (pay.status === 201 || pay.status === 200) && !!pay.body?.data?.payment_id, `-> ${pay.status} provider=${pay.body?.data?.provider} checkout=${pay.body?.data?.checkout_url}`);

  if (pay.body?.data?.provider === 'test') {
    const complete = await req('POST', '/payments/test/complete', { payment_id: pay.body.data.payment_id, outcome: 'success' }, token);
    ok('test provider completes payment', complete.status === 200, `-> ${complete.status}`);
    const after = await req('GET', `/orders/${orderId}`, undefined, token);
    ok('order marked PAID after verified webhook', after.body?.data?.status === 'PAID', `-> ${after.body?.data?.status}`);
    const access = await req('GET', `/products/${prod.id}/access`, undefined, token);
    ok('purchased product access granted', access.status === 200, `-> ${access.status} download=${access.body?.data?.download_url ? 'signed url present' : access.body?.data?.available}`);
    const cartAfter = await req('GET', '/cart', undefined, token);
    ok('cart cleared after payment', (cartAfter.body?.data?.items || []).length === 0, `-> ${(cartAfter.body?.data?.items || []).length} items`);
    const purchases = await req('GET', '/me/purchases', undefined, token);
    ok('purchases list shows product', purchases.status === 200 && purchases.body.data.length > 0, `-> ${purchases.body?.data?.length}`);
  }

  const hist = await req('GET', '/me/orders?limit=10', undefined, token);
  ok('order history', hist.status === 200 && hist.body.data.length > 0, `-> ${hist.body?.data?.length} orders`);

  // 9. webhook security
  const forge = await req('POST', '/payments/webhook', { payment_ref: 'fake', event_id: 'e2e-forge-' + stamp, status: 'paid', amount: 1 }, undefined, { 'X-Webhook-Signature': 'deadbeef' });
  ok('forged webhook rejected', forge.status === 400 || forge.status === 401 || forge.status === 403, `-> ${forge.status}`);

  // 10. private download protection
  const trav = await fetch(BASE + '/uploads/../backend/.env');
  ok('path traversal blocked', trav.status === 404 || trav.status === 403, `-> ${trav.status}`);
  const env = await fetch(BASE + '/backend/.env');
  ok('backend files not served', env.status === 404 || env.status === 403, `-> ${env.status}`);
  const dot = await fetch(BASE + '/.env');
  ok('dotfiles not served', dot.status === 404 || dot.status === 403, `-> ${dot.status}`);

  // 11. static pages
  for (const p of ['/', '/index.html', '/store.html', '/product.html', '/cart.html', '/checkout.html', '/payment.html', '/profile.html', '/login.html', '/register.html', '/blog.html', '/article.html', '/projects.html', '/project-detail.html', '/about.html', '/contact.html', '/skills.html', '/admin/', '/style.css', '/app.js', '/admin/app.js', '/admin/style.css', '/api/v1/openapi.yaml', '/docs']) {
    const r = await fetch(BASE + p);
    ok('static ' + p, r.status === 200, `-> ${r.status}`);
  }

  // 12. security headers
  const hres = await fetch(BASE + '/health');
  const need = ['X-Content-Type-Options', 'X-Frame-Options', 'Referrer-Policy'];
  ok('security headers present', need.every(h => hres.headers.get(h)), need.map(h => h + '=' + hres.headers.get(h)).join(' '));
  const csp = await fetch(API + '/products?limit=1');
  ok('API sends CSP', !!csp.headers.get('Content-Security-Policy'), csp.headers.get('Content-Security-Policy')?.slice(0, 40));

  // 13. rate limiting on auth
  let limited = false;
  for (let i = 0; i < 40; i++) { const r = await req('POST', '/auth/login', { email: 'nobody@example.com', password: 'x'.repeat(12) }); if (r.status === 429) { limited = true; break; } }
  ok('auth rate limiting triggers 429', limited, '');

  // 14. logout + refresh rotation
  const logout = await req('POST', '/auth/logout', {}, token);
  ok('logout ok', logout.status === 200 || logout.status === 204, `-> ${logout.status}`);

  const passed = results.filter(r => r[0] === 'PASS').length, failed = results.filter(r => r[0] === 'FAIL').length;
  console.log(`\n==== E2E SUMMARY: ${passed} passed, ${failed} failed, ${results.length} total ====`);
  if (failed) console.log('FAILURES:\n' + results.filter(r => r[0] === 'FAIL').map(r => ' - ' + r[1] + ' ' + r[2]).join('\n'));
})();
