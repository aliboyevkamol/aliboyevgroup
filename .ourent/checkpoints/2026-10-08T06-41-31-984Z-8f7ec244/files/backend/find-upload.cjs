const fs = require('fs');
const files = ['internal/api/admin.go', 'internal/api/misc.go', 'internal/api/server.go'];
const re = /upload|Upload|multipart|MaxBytes/;
for (const f of files) {
  const lines = fs.readFileSync(f, 'utf8').split(/\r?\n/);
  lines.forEach((l, i) => { if (re.test(l)) console.log(f + ':' + (i + 1) + ': ' + l.trim()); });
}
