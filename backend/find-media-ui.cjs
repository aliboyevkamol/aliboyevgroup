const fs = require('fs');
const f = '..\\admin\\app.js';
const lines = fs.readFileSync(f, 'utf8').split(/\r?\n/);
const re = /media|upload|cover_image|CoverImage|file|File|FormData|MULTIPART/;
lines.forEach((l, i) => { if (re.test(l)) console.log((i + 1) + ': ' + l.trim()); });
console.log('TOTAL_LINES=' + lines.length);
