// Measure a single login request time as the server sees it (client-side wall clock).
const API = 'http://localhost:8080/api/v1';
(async () => {
  const times = [];
  for (let i = 0; i < 5; i++) {
    const t0 = Date.now();
    const r = await fetch(API + '/auth/login', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: 'nobody@example.com', password: 'x'.repeat(12) }),
    });
    await r.text();
    times.push(Date.now() - t0);
  }
  console.log('login ms:', times.join(', '));
})();
