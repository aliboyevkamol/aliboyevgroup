// Inspect PNG chunks to verify the logo/favicon files.
const fs = require('fs');
for (const f of ['../logo.png', '../favicon.png']) {
  const s = fs.readFileSync(f);
  const sig = s.slice(0, 8).toString('hex');
  const w = s.readUInt32BE(16), h = s.readUInt32BE(20), bd = s[24], ct = s[25];
  const chunks = [];
  let o = 8;
  while (o < s.length - 8) {
    const len = s.readUInt32BE(o);
    const type = s.slice(o + 4, o + 8).toString('latin1');
    chunks.push(type + '(' + len + ')');
    o += 12 + len;
    if (type === 'IEND') break;
  }
  console.log(f, { sig, w, h, bitdepth: bd, colortype: ct, chunks: chunks.join(' '), sizeKB: Math.round(s.length / 1024) });
}
