const { execSync } = require('child_process');
function run(cmd){ try { return execSync(cmd, {encoding:'utf8', stdio:['ignore','pipe','pipe']}); } catch(e){ return 'ERR: '+(e.stderr||e.message); } }
console.log('--- tasklist docker ---');
console.log(run('tasklist /FI "IMAGENAME eq Docker Desktop.exe"'));
console.log('--- docker version ---');
console.log(run('docker version --format "{{.Server.Version}}"'));
console.log('--- fs ---');
console.log(run('dir "C:\\Program Files\\Docker\\Docker"'));
