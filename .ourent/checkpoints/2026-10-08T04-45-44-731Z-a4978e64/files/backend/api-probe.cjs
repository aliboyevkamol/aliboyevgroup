// Quick API probe: verifies the running container answers on the public API.
const BASE = 'http://localhost:8080';
const paths = ['/health', '/ready', '/api/v1/portfolio', '/api/v1/products?limit=3', '/api/v1/projects', '/api/v1/posts', '/index.html', '/store.html'];
(async () => {
  for (const p of paths) {
    try {
      const r = await fetch(BASE + p, { redirect: 'manual' });
      const t = await r.text();
      console.log(String(r.status).padEnd(4), p.padEnd(34), 'bytes=' + t.length, r.headers.get('content-type') || '');
    } catch (e) {
      console.log('ERR ', p, e.message);
    }
  }
})();
