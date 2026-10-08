// Probe: does auth rate limiting actually return 429?
const API = 'http://localhost:8080/api/v1';
(async () => {
  const codes = {};
  let first429 = -1;
  for (let i = 0; i < 40; i++) {
    const r = await fetch(API + '/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'nobody@example.com', password: 'x'.repeat(12) }),
    });
    codes[r.status] = (codes[r.status] || 0) + 1;
    if (r.status === 429 && first429 < 0) first429 = i;
  }
  console.log('status counts:', JSON.stringify(codes));
  console.log('first 429 at attempt index:', first429);
})();
