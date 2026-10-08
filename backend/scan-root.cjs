const fs = require('fs');
const path = require('path');
const root = path.join(__dirname, '..');
for (const f of fs.readdirSync(root)) {
  const s = fs.statSync(path.join(root, f));
  console.log((s.isDirectory() ? '[DIR] ' : String(s.size).padStart(8) + ' ') + JSON.stringify(f));
}
