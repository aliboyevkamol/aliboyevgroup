// helper: run a command and print output (avoids shell-quoting issues)
const { execSync } = require('child_process');
const cmd = process.argv.slice(2).join(' ');
try { process.stdout.write(execSync(cmd, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] })); }
catch (e) { process.stdout.write('ERR: ' + (e.stdout || '') + (e.stderr || '') + (e.message || '')); }
