// Replace the inline "KA" SVG favicon with the real ALIBOYEV GROUP icon set.
const fs = require('fs');
const path = require('path');

const HEAD_LINKS = `<link rel="icon" href="assets/favicon-32.png" sizes="32x32" type="image/png">
<link rel="icon" href="assets/favicon-64.png" sizes="64x64" type="image/png">
<link rel="icon" href="assets/favicon-192.png" sizes="192x192" type="image/png">
<link rel="apple-touch-icon" href="assets/apple-touch-icon.png" sizes="180x180">
<meta name="theme-color" content="#05070B">`;

const ADMIN_LINKS = HEAD_LINKS.replace(/assets\//g, '../assets/');

const pages = fs.readdirSync('.').filter(f => f.endsWith('.html'));
const oldIcon = /<link rel="icon" href="data:image\/svg\+xml,[^"]*">\r?\n?/;

let changed = [];
for (const p of pages) {
  let s = fs.readFileSync(p, 'utf8');
  if (!oldIcon.test(s)) { console.log('SKIP (no inline icon):', p); continue; }
  s = s.replace(oldIcon, HEAD_LINKS + '\n');
  fs.writeFileSync(p, s);
  changed.push(p);
}

// Admin panel: two levels deep, so the asset paths need "../".
const adminPath = path.join('admin', 'index.html');
let a = fs.readFileSync(adminPath, 'utf8');
if (oldIcon.test(a)) { a = a.replace(oldIcon, ADMIN_LINKS + '\n'); fs.writeFileSync(adminPath, a); changed.push(adminPath); }
else console.log('SKIP:', adminPath);

console.log('updated:', changed.join(', '));
