// Simulate the browser: after login the Set-Cookie must be accepted for a cross-site fetch.
// Node's fetch stores cookies only manually; here we just inspect the raw Set-Cookie attributes.
const BASE = 'http://localhost:8080/api/v1';
(async () => {
  const email = 'probe2_' + Date.now() + '@example.com';
  const pw = 'TestPassword123';
  let r = await fetch(BASE + '/auth/register', { method: 'POST', headers: { 'Content-Type': 'application/json', Origin: 'http://127.0.0.1:5500' }, body: JSON.stringify({ name: 'Probe Two', email, password: pw, confirm_password: pw }) });
  const sc = r.headers.getSetCookie ? r.headers.getSetCookie() : [r.headers.get('set-cookie')];
  console.log('Set-Cookie raw:', sc);
  // Login and reuse cookie
  r = await fetch(BASE + '/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json', Origin: 'http://127.0.0.1:5500' }, body: JSON.stringify({ email, password: pw }) });
  console.log('login set-cookie:', r.headers.getSetCookie ? r.headers.getSetCookie() : [r.headers.get('set-cookie')]);
})();
