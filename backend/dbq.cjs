// Tiny DB query helper: node dbq.cjs "select ..."  (talks to the postgres container)
const { execFileSync } = require('child_process');
const q = process.argv.slice(2).join(' ');
const out = execFileSync('docker', ['exec', 'backend-postgres-1', 'psql', '-U', 'kamol', '-d', 'kamol', '-A', '-F', '|', '-c', q.replace(/"/g, '')], { encoding: 'utf8' });
console.log(out.trim());
