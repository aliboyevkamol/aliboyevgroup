// Reproduce the 401 seen on /store.html after a user session existed, and show the exact request.
const { chromium } = require('playwright-core');
const BASE = 'http://localhost:8080';
const EDGE = 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe';
(async () => {
  const browser = await chromium.launch({ executablePath: EDGE, headless: true });
  const ctx = await browser.newContext();
  const p = await ctx.newPage();
  p.on('response', r => { if (r.status() >= 400) console.log('[http ' + r.status() + ']', r.request().method(), r.url(), '| type=' + r.request().resourceType()); });
  p.on('console', m => { if (m.type() === 'error') console.log('[console.error]', m.text()); });
  // register a user (creates the refresh cookie + session hint)
  await p.goto(BASE + '/register.html', { waitUntil: 'networkidle' });
  const email = 'probe_' + Date.now() + '@example.com';
  await p.fill('#nm', 'Probe User'); await p.fill('#em', email); await p.fill('#pw', 'StrongPass12345'); await p.fill('#pw2', 'StrongPass12345');
  await p.click('#rf button[type=submit]');
  await p.waitForURL(/profile\.html/, { timeout: 15000 }).catch(() => {});
  console.log('registered, url=' + p.url());
  // now delete the cookie to simulate an expired session while the hint remains
  await ctx.clearCookies();
  console.log('--- cookies cleared, navigating to /store.html ---');
  await p.goto(BASE + '/store.html', { waitUntil: 'networkidle' });
  await p.waitForTimeout(2000);
  await browser.close();
})();
