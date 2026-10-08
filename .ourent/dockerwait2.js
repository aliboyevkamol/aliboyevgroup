const { execSync } = require('child_process');
function run(cmd) {
  try { return { ok: true, out: execSync(cmd, { encoding: 'utf8', stdio: ['ignore','pipe','pipe'], timeout: 20000 }).trim() }; }
  catch (e) { return { ok: false, out: (String(e.stderr || e.stdout || e.message)).trim().slice(0, 200) }; }
}
(async () => {
  for (let i = 0; i < 18; i++) {
    const r = run('docker version --format {{.Server.Version}}');
    console.log(new Date().toISOString(), 'attempt', i + 1, r.ok ? 'READY ' + r.out : 'not-ready');
    if (r.ok) process.exit(0);
    await new Promise(res => setTimeout(res, 6000));
  }
  console.log('TIMEOUT: docker daemon never became ready');
})();
