const fs = require('fs');
const lines = fs.readFileSync('..\\app.js', 'utf8').split(/\r?\n/);
const re = /_srv\s*[:(]|function _srv|cover_image|media|upload/;
lines.forEach((l, i) => { if (re.test(l)) console.log((i + 1) + ': ' + l.trim()); });
