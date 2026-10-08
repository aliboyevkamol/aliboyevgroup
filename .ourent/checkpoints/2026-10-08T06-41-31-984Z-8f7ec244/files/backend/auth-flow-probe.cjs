// Quick probe: register -> login -> me -> logout -> login again, over 127.0.0.1:5500 origin.
const BASE = 'http://localhost:8080/api/v1';
const ORIGIN = 'http://127.0.0.1:5500';
const jar = [];
function setCookies(res) {
  const sc = res.headers.getSetCookie ? res.headers.getSetCookie() : [];
  for (const c of sc) { const [pair] = c.split(';'); const [k, v] = pair.split('='); const i = jar.findIndex(x => x.k === k); const o = { k, v }; if (i >= 0) jar[i] = o; else jar.push(o); }
}
const cookieHeader = () => jar.map(x => x.k + '=' + x.v).join('; ');
async function call(method, path, body, token) {
  const headers = { Origin: ORIGIN };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (token) headers.Authorization = 'Bearer ' + token;
  if (jar.length) headers.Cookie = cookieHeader();
  const r = await fetch(BASE + path, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) });
  setCookies(r);
  let j = null; try { j = await r.json(); } catch (e) {}
  return { status: r.status, ok: r.ok, body: j, acao: r.headers.get('access-control-allow-origin'), acc: r.headers.get('access-control-allow-credentials') };
}
(async () => {
  const email = 'probe' + Date.now() + '@example.com';
  const pw = 'TestPassword123';
  console.log('--- REGISTER ---');
  const reg = await call('POST', '/auth/register', { name: 'Probe User', email, password: pw, confirm_password: pw });
  console.log('status', reg.status, 'acao', reg.acao, 'acc', reg.acc);
  console.log('body', JSON.stringify(reg.body).slice(0, 300));
  const tok = reg.body && reg.body.data && reg.body.data.access_token;
  console.log('--- ME (with token) ---');
  const me = await call('GET', '/auth/me', undefined, tok);
  console.log('status', me.status, 'body', JSON.stringify(me.body).slice(0, 200));
  console.log('--- LOGOUT ---');
  const lo = await call('POST', '/auth/logout');
  console.log('status', lo.status);
  console.log('--- REFRESH after logout ---');
  const rf = await call('POST', '/auth/refresh');
  console.log('status', rf.status, 'body', JSON.stringify(rf.body).slice(0, 200));
  console.log('--- LOGIN again ---');
  const li = await call('POST', '/auth/login', { email, password: pw });
  console.log('status', li.status, 'body', JSON.stringify(li.body).slice(0, 300));
})();
