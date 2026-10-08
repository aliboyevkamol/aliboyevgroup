const fs = require('fs');
const base = __dirname + '/../';
for (const f of ['logo.png', 'favicon.png']) {
  const b = fs.readFileSync(base + f);
  const w = b.readUInt32BE(16), h = b.readUInt32BE(20);
  console.log(f, 'bytes=' + b.length, 'dims=' + w + 'x' + h);
}
