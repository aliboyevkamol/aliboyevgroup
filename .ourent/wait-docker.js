const { spawn, execSync } = require('child_process');
const exe = 'C:\\Program Files\\Docker\\Docker\\Docker Desktop.exe';
const p = spawn(exe, [], { detached: true, stdio: 'ignore' });
p.unref();
console.log('spawned pid', p.pid);
let tries = 0;
const iv = setInterval(() => {
  tries++;
  let out = '';
  try { out = execSync('docker version --format "{{.Server.Version}}"', {encoding:'utf8', stdio:['ignore','pipe','pipe']}).trim(); } catch(e){ out = 'not-ready'; }
  console.log(Date.now(), 'try', tries, '->', out);
  if (out !== 'not-ready' || tries >= 40) { clearInterval(iv); process.exit(0); }
}, 5000);
