const fs = require('fs');
const files = ['internal/api/catalog.go', 'internal/api/uploads.go'];
for (const f of files) {
  if (!fs.existsSync(f)) { console.log('MISSING ' + f); continue; }
  const lines = fs.readFileSync(f, 'utf8').split(/\r?\n/);
  lines.forEach((l, i) => { if (/cover_image|CoverImage/.test(l)) console.log(f + ':' + (i + 1) + ': ' + l.trim()); });
}
