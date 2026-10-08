// Replace the inline "KA" SVG favicon with the real ALIBOYEV GROUP icon set.
// Run from the project root:  node backend/_tools/patch-favicon.cjs   (see run instructions below)
const fs = require('fs');
const path = require('path');

const ROOT = process.cwd();

const HEAD_LINKS = `<link rel="icon" href="assets/favicon-32.png" sizes="32x32" type="image/png">
<link rel="icon" href="assets/favicon-64.png" sizes="64x64" type="image/png">
<link rel="icon" href="assets/favicon-192.png" sizes="192x192" type="image/png">
<link rel="apple-touch-icon" href="assets/apple-touch-icon.png" sizes="180x180">
<meta name="theme-color" content="#05070B">`;

const ADMIN_LINKS = HEAD_LINKS.replace(/assets\//g, '../assets/');

const pages = fs.readdirSync(ROOT).filter(f => f.endsWith('.html'));
const oldIcon = /<link rel="icon" href="data:image\/svg\+xml,[^"]*">\r?\n?/;

const changed = [];
for (const p of pages) {
  const full = path.join(ROOT, p);
  let s = fs.readFileSync(full, 'utf8');
  if (!oldIcon.test(s)) { console.log('SKIP (no inline icon):', p); continue; }
  s = s.replace(oldIcon, HEAD_LINKS + '\n');
  s = s.replace(/<meta name="theme-color"[^>]*>\r?\n/g, '');
  fs.writeFileSync(full, s);
  changed.push(p);
}

const adminPath = path.join(ROOT, 'admin', 'index.html');
let a = fs.readFileSync(adminPath, 'utf8');
if (oldIcon.test(a)) {
  a = a.replace(oldIcon, ADMIN_LINKS + '\n');
  a = a.replace(/<meta name="theme-color"[^>]*>\r?\n/g, '');
  fs.writeFileSync(adminPath, a);
  changed.push('admin/index.html');
} else console.log('SKIP: admin/index.html');

console.log('updated ' + changed.length + ' files:', changed.join(', '));
