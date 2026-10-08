// Real-browser test of every page: console errors, network failures, DOM rendering.
const { chromium } = require('playwright-core');
const BASE = 'http://localhost:8080';
const EDGE = 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe';
const pages = [
  ['/', 'index'], ['/about.html', 'about'], ['/skills.html', 'skills'], ['/projects.html', 'projects'],
  ['/project-detail.html?project=digital-school', 'project-detail'], ['/store.html', 'store'],
  ['/product.html?id=1', 'product'], ['/blog.html', 'blog'], ['/article.html?id=1', 'article'],
  ['/contact.html', 'contact'], ['/cart.html', 'cart'], ['/checkout.html', 'checkout'],
  ['/payment.html?order=1', 'payment'], ['/login.html', 'login'], ['/register.html', 'register'],
  ['/profile.html', 'profile'], ['/admin/', 'admin']
];
const results = [];
const ok = (n, c, extra = '') => { results.push([c ? 'PASS' : 'FAIL', n, extra]); console.log(`${c ? 'PASS' : 'FAIL'} | ${n} ${extra}`); };

(async () => {
  const browser = await chromium.launch({ executablePath: EDGE, headless: true });
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });

  const consoleErrors = [], pageErrors = [], failedRequests = [], httpErrors = [];
  ctx.on('page', p => {
    p.on('console', m => { if (m.type() === 'error') consoleErrors.push(`${p.url()} :: ${m.text().slice(0, 160)}`); });
    p.on('pageerror', e => pageErrors.push(`${p.url()} :: ${String(e).slice(0, 160)}`));
    p.on('requestfailed', r => failedRequests.push(`${r.url()} :: ${r.failure()?.errorText}`));
    p.on('response', r => { if (r.status() >= 400) httpErrors.push(`${r.status()} ${r.request().method()} ${r.url()} (type=${r.request().resourceType()}) from ${p.url()}`); });
  });

  for (const [path, name] of pages) {
    const page = await ctx.newPage();
    const resp = await page.goto(BASE + path, { waitUntil: 'networkidle', timeout: 30000 }).catch(e => null);
    await page.waitForTimeout(700);
    const status = resp ? resp.status() : 0;
    const empty = await page.evaluate(() => !document.body.innerText.trim().length);
    const nav = await page.evaluate(() => {
      const h = document.querySelector('header.nav');
      return { header: !!h, links: h ? h.querySelectorAll('.links a').length : 0, footer: !!document.querySelector('footer.foot') };
    });
    ok(`browser ${path}`, status === 200 && !empty, `-> ${status}, header=${nav.header} links=${nav.links} footer=${nav.footer}`);

    // Page-specific rendering checks (content must come from the live API)
    const checks = {
      index: async () => ({ fp: await page.locator('#fp .card').count(), fs: await page.locator('#fs .card').count() }),
      store: async () => ({ products: await page.locator('#sg .card').count(), pills: await page.locator('#pills .pill').count() }),
      skills: async () => ({ bars: await page.locator('#skg .sk').count(), pills: await page.locator('#pills .pill').count() }),
      projects: async () => ({ projects: await page.locator('#pl .card').count() }),
      blog: async () => ({ posts: await page.locator('#bl .card').count() }),
      'project-detail': async () => ({ title: await page.locator('#pd h1').count(), discussion: await page.locator('.disc').count(), flow: await page.locator('.flow li').count() }),
      product: async () => ({ title: await page.locator('#pr h1').count(), buy: await page.locator('.buy .btn.pri').count(), related: await page.locator('#pr .sg .card').count() }),
      article: async () => ({ title: await page.locator('#ar h1').count(), discussion: await page.locator('.disc').count() })
    };
    if (checks[name]) {
      const r = await checks[name]();
      const good = Object.values(r).every(v => v > 0);
      ok(`browser content ${name}`, good, JSON.stringify(r));
    }
    await page.close();
  }

  // ---- interactive flows ----
  // 1. theme + language toggle persist
  const p1 = await ctx.newPage();
  await p1.goto(BASE + '/index.html', { waitUntil: 'networkidle' });
  const theme0 = await p1.getAttribute('html', 'data-theme');
  await p1.click('#theme');
  const theme1 = await p1.getAttribute('html', 'data-theme');
  await p1.click('#lang');
  const lang1 = await p1.textContent('#lang');
  await p1.click('#lang');
  const navText = (await p1.locator('.links a').first().textContent()).trim();
  ok('theme toggle works', theme0 !== theme1, `${theme0} -> ${theme1}`);
  ok('language toggle works', navText.length > 0 && lang1 !== 'EN', `nav="${navText}"`);
  await p1.close();

  // 2. guest cart: add to cart -> badge -> cart page -> checkout gate
  const p2 = await ctx.newPage();
  await p2.goto(BASE + '/store.html', { waitUntil: 'networkidle' });
  await p2.waitForTimeout(500);
  const storeCount = await p2.locator('#sg .card').count();
  await p2.locator('#sg .card').first().locator('[data-cart]').click();
  await p2.waitForTimeout(400);
  const badge = (await p2.textContent('.cb')).trim();
  ok('add to cart updates badge', badge === '1', `badge=${badge}, storeCards=${storeCount}`);
  await p2.goto(BASE + '/cart.html', { waitUntil: 'networkidle' });
  await p2.waitForTimeout(500);
  const rows = await p2.locator('.ci').count();
  const total = (await p2.textContent('.sum .tot b')).trim();
  ok('cart page lists item + total', rows === 1 && /^\$\d/.test(total), `rows=${rows} total=${total}`);
  await p2.click('[data-inc]');
  await p2.waitForTimeout(300);
  const qty = (await p2.textContent('.qty span')).trim();
  ok('cart quantity increase', qty === '2', `qty=${qty}`);
  await p2.goto(BASE + '/checkout.html', { waitUntil: 'networkidle' });
  await p2.waitForTimeout(500);
  const gate = await p2.locator('text=Please log in to continue').count();
  ok('checkout requires login', gate > 0, '');
  await p2.close();

  // 3. store search + sort + category filter
  const p3 = await ctx.newPage();
  await p3.goto(BASE + '/store.html', { waitUntil: 'networkidle' });
  await p3.waitForTimeout(500);
  const all = await p3.locator('#sg .card').count();
  await p3.fill('#q', 'telegram');
  await p3.waitForTimeout(300);
  const searched = await p3.locator('#sg .card').count();
  ok('store search filters products', searched > 0 && searched < all, `all=${all} searched=${searched}`);
  await p3.fill('#q', 'zzzznothing');
  await p3.waitForTimeout(300);
  const emptyVisible = await p3.locator('#empty').isVisible();
  ok('store empty state shown', emptyVisible, '');
  await p3.fill('#q', '');
  await p3.selectOption('#so', 'lo');
  await p3.waitForTimeout(300);
  const prices = await p3.locator('#sg .pb b').allTextContents();
  const nums = prices.map(t => parseFloat(String(t).replace(/[^0-9.]/g, '')));
  const sorted = nums.every((v, i) => i === 0 || nums[i - 1] <= v);
  ok('store price sort ascending', sorted, nums.join(','));
  await p3.close();

  // 4. buy now from product page (guest) -> cart + redirect to checkout
  const p4 = await ctx.newPage();
  await p4.goto(BASE + '/product.html?id=2', { waitUntil: 'networkidle' });
  await p4.waitForTimeout(500);
  const t = (await p4.textContent('#pr h1')).trim();
  await p4.click('.buy [data-buy]');
  await p4.waitForTimeout(1200);
  ok('buy now navigates to checkout', p4.url().includes('checkout.html'), `url=${p4.url()} product="${t}"`);
  await p4.close();

  // 5. registration + login + profile in the browser
  const p5 = await ctx.newPage();
  const email = `browser_${Date.now()}@example.com`;
  await p5.goto(BASE + '/register.html', { waitUntil: 'networkidle' });
  await p5.fill('#nm', 'Browser Tester');
  await p5.fill('#em', email);
  await p5.fill('#pw', 'StrongPass12345');
  await p5.fill('#pw2', 'StrongPass12345');
  await p5.click('#rf button[type=submit]');
  await p5.waitForURL(/profile\.html/, { timeout: 15000 }).catch(() => {});
  ok('register -> profile redirect', p5.url().includes('profile.html'), `url=${p5.url()}`);
  await p5.waitForTimeout(900);
  const profileName = await p5.textContent('.pfh h1').catch(() => '');
  ok('profile shows user', /Browser Tester/.test(profileName), `name="${profileName.trim()}"`);
  // favorites from a product card
  await p5.goto(BASE + '/store.html', { waitUntil: 'networkidle' });
  await p5.waitForTimeout(600);
  await p5.locator('#sg .card').first().locator('[data-fav]').click();
  await p5.waitForTimeout(600);
  await p5.goto(BASE + '/profile.html', { waitUntil: 'networkidle' });
  await p5.waitForTimeout(900);
  const favCards = await p5.locator('#pf .sg .card').count();
  ok('favorite saved to profile', favCards > 0, `favCards=${favCards}`);
  await p5.close();

  // 6. full purchase through the real UI (card form -> test provider -> paid)
  // Use a fresh browser context so the previous user's session can't redirect register -> profile.
  const buyCtx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  buyCtx.on('page', p => {
    p.on('console', m => { if (m.type() === 'error') consoleErrors.push(`${p.url()} :: ${m.text().slice(0, 160)}`); });
    p.on('pageerror', e => pageErrors.push(`${p.url()} :: ${String(e).slice(0, 160)}`));
    p.on('requestfailed', r => failedRequests.push(`${r.url()} :: ${r.failure()?.errorText}`));
    p.on('response', r => { if (r.status() >= 400) httpErrors.push(`${r.status()} ${r.request().method()} ${r.url()} (type=${r.request().resourceType()}) from ${p.url()}`); });
  });
  const p6 = await buyCtx.newPage();
  const email2 = `buyer_${Date.now()}@example.com`;
  await p6.goto(BASE + '/register.html', { waitUntil: 'networkidle' });
  await p6.fill('#nm', 'Buyer Tester'); await p6.fill('#em', email2); await p6.fill('#pw', 'StrongPass12345'); await p6.fill('#pw2', 'StrongPass12345');
  await p6.click('#rf button[type=submit]');
  await p6.waitForURL(/profile\.html/, { timeout: 15000 }).catch(() => {});
  await p6.goto(BASE + '/store.html', { waitUntil: 'networkidle' }); await p6.waitForTimeout(500);
  await p6.locator('#sg .card').first().locator('[data-buy]').click();
  await p6.waitForURL(/checkout\.html/, { timeout: 15000 }).catch(() => {});
  await p6.waitForTimeout(800);
  const coFields = await p6.locator('#cof').count();
  if (coFields) {
    await p6.fill('#fn', 'Buyer Tester'); await p6.fill('#fe', email2); await p6.fill('#fp', '+998901112233'); await p6.fill('#fc', 'Uzbekistan');
    await p6.click('#cb');
    await p6.waitForURL(/payment\.html/, { timeout: 20000 }).catch(() => {});
    await p6.waitForTimeout(800);
    ok('checkout creates order -> payment page', p6.url().includes('payment.html'), `url=${p6.url()}`);
    // card form validation first (bad card must be rejected client-side)
    await p6.fill('#cn', 'Buyer Tester'); await p6.fill('#cc', '1234'); await p6.fill('#ce', '13/99'); await p6.fill('#cv', '12');
    await p6.click('#pb');
    await p6.waitForTimeout(400);
    const err1 = (await p6.textContent('#err')).trim();
    ok('invalid card rejected in browser', /card details/i.test(err1), `err="${err1}"`);
    // valid card form
    await p6.fill('#cc', '4242424242424242'); await p6.fill('#ce', '12/30'); await p6.fill('#cv', '123');
    await p6.click('#pb');
    await p6.waitForTimeout(1500);
    const success = await p6.locator('text=Payment successful').count();
    ok('test payment completes in UI', success > 0, `url=${p6.url()}`);
    const accessRow = await p6.locator('.card.cform .ln').count();
    ok('payment page shows product access', accessRow > 0, `rows=${accessRow}`);
  } else {
    ok('checkout creates order -> payment page', false, 'checkout form missing');
  }
  await p6.close();
  await buyCtx.close();

  // 7. admin panel in the browser
  const p7 = await ctx.newPage();
  await p7.goto(BASE + '/admin/', { waitUntil: 'networkidle' });
  await p7.fill('#ae', 'admin@local.dev'); await p7.fill('#ap', 'admin_password_12345');
  await p7.click('#lf button[type=submit]');
  // The admin panel boots its data after login; wait for the panel to actually open (was a fixed 2.5s).
  await p7.locator('#app').waitFor({ state: 'visible', timeout: 20000 }).catch(() => {});
  await p7.waitForTimeout(600);
  const appVisible = await p7.locator('#app').isVisible().catch(() => false);
  const titles = (await p7.locator('#nav a, #nav button').allTextContents()).length;
  ok('admin login shows panel', appVisible && titles > 0, `panelVisible=${appVisible} navItems=${titles}`);
  const adminRows = await p7.locator('#view tr, #view .card').count();
  ok('admin dashboard renders data', adminRows > 0, `rows=${adminRows}`);
  await p7.close();

  await browser.close();

  const realConsoleErrors = consoleErrors.filter(e => !/favicon/i.test(e));
  const realFailed = failedRequests.filter(r => !/favicon/i.test(r));
  const realHttp = httpErrors.filter(e => !/favicon/i.test(e));
  ok('no uncaught page errors', pageErrors.length === 0, pageErrors.slice(0, 3).join(' | '));
  ok('no console errors', realConsoleErrors.length === 0, realConsoleErrors.slice(0, 3).join(' | '));
  ok('no failed network requests', realFailed.length === 0, realFailed.slice(0, 3).join(' | '));

  const passed = results.filter(r => r[0] === 'PASS').length, failed = results.filter(r => r[0] === 'FAIL').length;
  console.log(`\n==== BROWSER E2E: ${passed} passed, ${failed} failed, ${results.length} total ====`);
  if (failed) console.log('FAILURES:\n' + results.filter(r => r[0] === 'FAIL').map(r => ' - ' + r[1] + ' ' + r[2]).join('\n'));
  console.log('\nconsole errors:', realConsoleErrors.length, '| page errors:', pageErrors.length, '| failed requests:', realFailed.length);
  if (realConsoleErrors.length) console.log('CONSOLE:\n' + realConsoleErrors.slice(0, 10).join('\n'));
  if (realFailed.length) console.log('NETFAIL:\n' + realFailed.slice(0, 10).join('\n'));
  if (realHttp.length) console.log('HTTP>=400 (' + realHttp.length + '):\n' + [...new Set(realHttp)].slice(0, 12).join('\n'));
})();
