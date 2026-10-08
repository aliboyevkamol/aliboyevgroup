// Probe: does POST /auth/login hit 429 after many attempts?
const BASE = 'http://localhost:8080/api/v1';
(async () => {
  let codes = {};
  for (let i = 0; i < 45; i++) {
    const r = await fetch(BASE + '/auth/login', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'nobody@example.com', password: 'x'.repeat(12) })
    });
    codes[r.status] = (codes[r.status] || 0) + 1;
    if (r.status === 429) { console.log('429 first seen at attempt', i + 1); break; }
  }
  console.log('status counts:', JSON.stringify(codes));
})();
