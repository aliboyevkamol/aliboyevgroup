// Real auth flow + CORS probe against the running backend.
const BASE = 'http://localhost:8080/api/v1';
const ORIGIN = 'http://127.0.0.1:5500';

const cookies = {};
function cookieHeader() { return Object.entries(cookies).map(([k, v]) => k + '=' + v).join('; '); }
function storeCookies(r) {
  const raw = r.headers.getSetCookie ? r.headers.getSetCookie() : [];
  for (const c of raw) { const [pair] = c.split(';'); const i = pair.indexOf('='); cookies[pair.slice(0, i)] = pair.slice(i + 1); }
}
async function call(method, path, body, extra = {}) {
  const headers = { Origin: ORIGIN, ...(extra.headers || {}) };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  const ck = cookieHeader(); if (ck) headers.Cookie = ck;
  const r = await fetch(BASE + path, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) });
  storeCookies(r);
  let j = null; try { j = await r.json(); } catch (e) {}
  return { status: r.status, headers: r.headers, json: j };
}
const line = (t, ok, extra = '') => console.log((ok ? 'PASS' : 'FAIL') + ' | ' + t + (extra ? ' | ' + extra : ''));

(async () => {
  const email = 'probe+' + Date.now() + '@example.com';
  const pass = 'TestPass12345';

  // 1. CORS preflight from the dev frontend origin
  const pre = await fetch(BASE + '/auth/login', { method: 'OPTIONS', headers: { Origin: ORIGIN, 'Access-Control-Request-Method': 'POST', 'Access-Control-Request-Headers': 'content-type' } });
  line('CORS preflight allows 127.0.0.1:5500', pre.headers.get('access-control-allow-origin') === ORIGIN && pre.headers.get('access-control-allow-credentials') === 'true',
    'acao=' + pre.headers.get('access-control-allow-origin'));

  // 2. register
  const reg = await call('POST', '/auth/register', { name: 'Probe User', email, password: pass, confirm_password: pass });
  line('register returns 200 + access token', reg.status === 200 && !!reg.json?.data?.access_token, 'status=' + reg.status);
  const rt = reg.headers.get('set-cookie') || reg.headers.getSetCookie?.().join(';') || '';
  line('refresh cookie set with SameSite/Secure', /kamol_rt=/.test(cookieHeader() ? 'kamol_rt=' : '') && /samesite=none/i.test(rt) && /secure/i.test(rt), rt.replace(/kamol_rt=[^;]+/, 'kamol_rt=***'));
  const token = reg.json?.data?.access_token;

  // 3. me with the access token
  const me = await call('GET', '/auth/me', undefined, { headers: { Authorization: 'Bearer ' + token } });
  line('GET /auth/me with token', me.status === 200 && me.json?.data?.email === email, 'status=' + me.status);

  // 4. refresh using only the cookie (simulates a page reload)
  const ref = await call('POST', '/auth/refresh');
  line('refresh with cookie only (cross-origin)', ref.status === 200 && !!ref.json?.data?.access_token, 'status=' + ref.status);

  // 5. logout
  const out = await call('POST', '/auth/logout');
  line('logout returns 200', out.status === 200, 'status=' + out.status);

  // 6. refresh after logout must fail
  const ref2 = await call('POST', '/auth/refresh');
  line('refresh after logout is rejected', ref2.status === 401, 'status=' + ref2.status);

  // 7. login again
  const log = await call('POST', '/auth/login', { email, password: pass });
  line('login after logout works', log.status === 200 && !!log.json?.data?.access_token, 'status=' + log.status);

  // 8. wrong password rejected
  const bad = await call('POST', '/auth/login', { email, password: 'wrong-password-1' });
  line('wrong password rejected 401', bad.status === 401, 'status=' + bad.status);

  // 9. disallowed origin gets no CORS header
  const badOrigin = await fetch(BASE + '/products', { headers: { Origin: 'http://evil.example' } });
  line('unknown origin not allowed', badOrigin.headers.get('access-control-allow-origin') === null, 'acao=' + badOrigin.headers.get('access-control-allow-origin'));

  // 10. guest cart write rejected (401) and admin media rejected
  const gcart = await call('POST', '/cart/items', { product_id: 1, quantity: 1 });
  line('guest cart write returns 401 (expected)', gcart.status === 401, 'status=' + gcart.status);
})().catch(e => { console.error('probe crashed: ' + e.message); process.exit(1); });
