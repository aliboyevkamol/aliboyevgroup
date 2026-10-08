// Probe: inspect every response for Retry-After / X-* to understand limiter behaviour.
const API = 'http://localhost:8080/api/v1';
(async () => {
  for (let i = 0; i < 15; i++) {
    const r = await fetch(API + '/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Real-IP': '10.9.9.9' },
      body: JSON.stringify({ email: 'nobody@example.com', password: 'x'.repeat(12) }),
    });
    console.log(i, r.status, 'retry-after=', r.headers.get('retry-after'), 'xrid=', r.headers.get('x-request-id'));
  }
})();
