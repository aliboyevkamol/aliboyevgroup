const fs = require('fs');
const lines = fs.readFileSync('e2e-run.log', 'utf8').split(/\r?\n/).filter(Boolean);
console.log('TOTAL LINES:', lines.length);
console.log(lines.slice(-20).join('\n'));
