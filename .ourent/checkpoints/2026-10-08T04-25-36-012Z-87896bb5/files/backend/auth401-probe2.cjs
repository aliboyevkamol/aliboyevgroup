// Why does the browser test see "Failed to load resource: 401" on store/cart/product?
// Hypothesis: a stale session hint triggers POST /auth/refresh -> 401 (expected for a guest).
const { chromium } = require('playwright-core');
const BASE = 'http://localhost:8080';
const EDGE = 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe';
(async () => {
  const browser = await chromium.launch({ executablePath: EDGE, headless: true });
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  ctx.on('page', p => {
    p.on('console', m => { if (m.type() === 'error') console.log('[console.error]', p.url(), '::', m.text().slice(0, 140)); });
    p.on('response', r => { if (r.status() >= 400) console.log('[http ' + r.status() + ']', r.request().method(), r.url(), '(type=' + r.request().resourceType() + ')'); });
    p.on('request', r => { if (/auth\/(refresh|me|logout)/.test(r.url())) console.log('[req]', r.method(), r.url()); });
  });

  // simulate a stale hint: set sessionStorage flag WITHOUT a valid refresh cookie
  const p1 = await ctx.newPage();
  await p1.goto(BASE + '/index.html', { waitUntil: 'networkidle' });
  await p1.evaluate(() => { sessionStorage.setItem('kamol_has_session', '1'); });
  await p1.close();

  console.log('\n--- opening store.html with a stale session hint ---');
  const p2 = await ctx.newPage();
  await p2.goto(BASE + '/store.html', { waitUntil: 'networkidle' });
  await p2.waitForTimeout(1500);
  console.log('store products =', await p2.locator('#sg .card').count());
  await p2.close();

  await browser.close();
})();
