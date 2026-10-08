const fs = require('fs');
const path = require('path');
const root = path.join(__dirname, '..');
const junk = ['$null', 'ids.indexOf(v)!', 'tb.bat', 'run-gotest-root.bat', 'check-pages-root.cjs', 'patch-favicon.cjs'];
let removed = [];
for (const f of junk) {
  const p = path.join(root, f);
  try { if (fs.existsSync(p)) { fs.unlinkSync(p); removed.push(f); } } catch (e) { console.log('FAIL', f, e.message); }
}
console.log('Removed:', removed.join(', ') || '(none)');
