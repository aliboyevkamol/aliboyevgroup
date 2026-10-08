const fs = require('fs');
const path = require('path');
const root = path.join(__dirname, '..');
const files = fs.readdirSync(root).filter(f => f.endsWith('.html')).concat(['admin/index.html']).concat(fs.readdirSync(path.join(root, 'admin')).filter(f => f.endsWith('.html')).map(f => 'admin/' + f));
for (const f of files) {
  const p = path.join(root, f);
  if (!fs.existsSync(p)) { console.log(f, 'MISSING'); continue; }
  const s = fs.readFileSync(p, 'utf8');
  const icons = (s.match(/<link[^>]*rel=["'](?:icon|apple-touch-icon)[^>]*>/gi) || []);
  const logos = (s.match(/assets\/logo[^"')\s]*/gi) || []);
  console.log(f + ' | icons=' + icons.length + ' logos=' + logos.length);
  if (icons.length) console.log('    ', icons.join('\n     '));
}
