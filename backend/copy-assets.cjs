const fs = require('fs');
fs.mkdirSync('../assets', { recursive: true });
for (const f of fs.readdirSync('assets')) fs.copyFileSync('assets/' + f, '../assets/' + f);
console.log('assets:', fs.readdirSync('../assets').join(', '));
