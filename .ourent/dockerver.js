const { execSync } = require('child_process');
function run(cmd) {
  try { return { ok: true, out: execSync(cmd, { encoding: 'utf8', stdio: ['ignore','pipe','pipe'], timeout: 20000 }).trim() }; }
  catch (e) { return { ok: false, out: (String(e.stderr || e.stdout || e.message)).trim().slice(0, 400) }; }
}
const r = run('docker version --format {{.Server.Version}}');
console.log(r.ok ? 'DOCKER_DAEMON_VERSION=' + r.out : 'NOT_READY: ' + r.out);
