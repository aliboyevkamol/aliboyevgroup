// Find the exact URL behind the console 404 on /store.html.
const { chromium } = require('playwright-core');
const BASE = 'http://localhost:8080';
const EDGE = 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe';
(async () => {
  const browser = await chromium.launch({ executablePath: EDGE, headless: true });
  const ctx = await browser.newContext();
  const p = await ctx.newPage();
  p.on('request', r => console.log('[req]', r.method(), r.url(), '| type=' + r.resourceType()));
  p.on('response', r => { if (r.status() >= 400) console.log('[RESP ' + r.status() + ']', r.url(), '| type=' + r.request().resourceType()); });
  p.on('console', m => { if (m.type() === 'error') console.log('[console.error]', m.text()); });
  p.on('requestfailed', r => console.log('[failed]', r.url(), r.failure()?.errorText, '| type=' + r.resourceType()));
  await p.goto(BASE + '/store.html', { waitUntil: 'networkidle', timeout: 30000 });
  await p.waitForTimeout(2000);
  await browser.close();
})();
