// Admin panel in a clean context: login -> panel -> dashboard data.
const { chromium } = require('playwright-core');
const BASE = 'http://localhost:8080';
const EDGE = 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe';
(async () => {
  const browser = await chromium.launch({ executablePath: EDGE, headless: true });
  const ctx = await browser.newContext();
  const p = await ctx.newPage();
  p.on('console', m => { if (m.type() === 'error') console.log('[console.error]', m.text()); });
  p.on('response', r => { if (r.status() >= 400) console.log('[http ' + r.status() + ']', r.request().method(), r.url()); });
  await p.goto(BASE + '/admin/', { waitUntil: 'networkidle' });
  await p.fill('#ae', 'admin@local.dev');
  await p.fill('#ap', 'admin_password_12345');
  await p.click('#lf button[type=submit]');
  for (let i = 0; i < 12; i++) {
    await p.waitForTimeout(1000);
    const vis = await p.locator('#app').isVisible().catch(() => false);
    const err = await p.locator('#lerr').textContent().catch(() => '');
    console.log('t=' + (i + 1) + 's appVisible=' + vis + ' loginError="' + err + '"');
    if (vis) break;
  }
  const rows = await p.locator('#view tr, #view .card').count();
  const nav = await p.locator('#nav a').count();
  console.log('FINAL appVisible=' + await p.locator('#app').isVisible().catch(() => false), 'navItems=' + nav, 'dashboardRows=' + rows);
  await browser.close();
})();
