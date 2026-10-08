// Reproduce the browser-test admin step inside a SHARED context (which already
// holds a logged-in, non-admin user's refresh cookie) to find why #app stays hidden.
const { chromium } = require('playwright-core');
const BASE = 'http://localhost:8080';
const EDGE = 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe';
(async () => {
  const browser = await chromium.launch({ executablePath: EDGE, headless: true });
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  ctx.on('page', p => {
    p.on('console', m => { if (m.type() === 'error') console.log('[console.error]', m.text().slice(0, 160)); });
    p.on('response', r => { if (r.status() >= 400) console.log('[http ' + r.status() + ']', r.request().method(), r.url()); });
  });

  // 1. register a normal user in this same context (like the browser test does)
  const p1 = await ctx.newPage();
  await p1.goto(BASE + '/register.html', { waitUntil: 'networkidle' });
  await p1.fill('#nm', 'Shared User');
  await p1.fill('#em', 'shared' + Date.now() + '@test.dev');
  await p1.fill('#pw', 'password_12345');
  await p1.click('#rf button[type=submit]').catch(() => {});
  await p1.waitForTimeout(2500);
  console.log('after register url =', p1.url());
  await p1.close();

  // 2. now open the admin panel in the same context
  const p7 = await ctx.newPage();
  await p7.goto(BASE + '/admin/', { waitUntil: 'networkidle' });
  await p7.waitForTimeout(1000);
  console.log('pre-login appVisible =', await p7.locator('#app').isVisible().catch(() => false));
  await p7.fill('#ae', 'admin@local.dev');
  await p7.fill('#ap', 'admin_password_12345');
  await p7.click('#lf button[type=submit]');
  for (let i = 0; i < 15; i++) {
    await p7.waitForTimeout(1000);
    const vis = await p7.locator('#app').isVisible().catch(() => false);
    const err = await p7.locator('#lerr').textContent().catch(() => '');
    console.log('t=' + (i + 1) + 's appVisible=' + vis + ' loginError="' + err + '"');
    if (vis) break;
  }
  const rows = await p7.locator('#view tr, #view .card').count();
  console.log('FINAL appVisible=' + await p7.locator('#app').isVisible().catch(() => false), 'rows=' + rows);
  await browser.close();
})();
