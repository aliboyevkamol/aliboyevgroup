// Finder: locate an installed Chromium-based browser or a Playwright browser.
const fs = require('fs');
const candidates = [
  'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
  'C:/Program Files/Microsoft/Edge/Application/msedge.exe',
  'C:/Program Files/Google/Chrome/Application/chrome.exe',
  'C:/Program Files (x86)/Google/Chrome/Application/chrome.exe',
];
console.log('Browsers:');
for (const c of candidates) console.log((fs.existsSync(c) ? 'FOUND ' : 'MISS  ') + c);
const pw = 'C:/Users/Kamol/AppData/Local/ms-playwright';
console.log('\nms-playwright:', fs.existsSync(pw) ? fs.readdirSync(pw).join(', ') : 'NOT FOUND');
try {
  const { chromium } = require('playwright-core');
  console.log('playwright-core version:', require('playwright-core/package.json').version);
  console.log('chromium executablePath:', chromium.executablePath());
  console.log('chromium exists:', fs.existsSync(chromium.executablePath()));
} catch (e) { console.log('playwright-core error:', e.message); }
