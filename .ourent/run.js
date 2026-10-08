const { execSync } = require('child_process');
function run(cmd) {
  try { return { ok: true, out: execSync(cmd, { encoding: 'utf8', stdio: ['ignore','pipe','pipe'], timeout: 15000 }).trim() }; }
  catch (e) { return { ok: false, out: (String(e.stderr || e.stdout || e.message)).trim().slice(0, 300) }; }
}
const cmds = process.argv.slice(2);
for (const c of cmds) {
  const r = run(c);
  console.log('=== ' + c + ' => ' + (r.ok ? 'OK' : 'FAIL'));
  console.log(r.out);
}
