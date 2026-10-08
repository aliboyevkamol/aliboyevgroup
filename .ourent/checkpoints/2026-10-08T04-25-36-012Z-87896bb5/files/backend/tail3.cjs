const fs = require('fs');
const file = process.argv[2] || 'browser-test-3.log';
const n = Number(process.argv[3] || 40);
try {
  const lines = fs.readFileSync(file, 'utf8').split(/\r?\n/);
  console.log('TOTAL LINES:', lines.length);
  console.log(lines.slice(-n).join('\n'));
} catch (e) { console.log('no log yet:', e.message); }
