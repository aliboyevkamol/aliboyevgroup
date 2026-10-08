const fs = require('fs');
const lines = fs.readFileSync('e2e-run.log', 'utf8').split(/\r?\n/).filter(Boolean);
const pass = lines.filter(l => l.startsWith('PASS')).length;
const fail = lines.filter(l => l.startsWith('FAIL')).length;
console.log('E2E summary: PASS=' + pass + ' FAIL=' + fail + ' lines=' + lines.length);
console.log('\n--- ALL FAILURES ---');
lines.filter(l => l.startsWith('FAIL')).forEach(l => console.log(l));
