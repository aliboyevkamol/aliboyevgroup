const fs = require('fs');
const s = fs.readFileSync('../app.js', 'utf8');
const i = s.indexOf('_srv');
console.log('first _srv index:', i);
const seg = s.slice(i - 2, i + 80);
console.log(JSON.stringify(seg));
console.log('codes:', [...seg].map(c => c.charCodeAt(0)).join(','));
