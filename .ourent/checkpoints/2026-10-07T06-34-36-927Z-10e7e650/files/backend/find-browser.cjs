const fs = require('fs');
const p = ['C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe', 'C:/Program Files/Google/Chrome/Application/chrome.exe', 'C:/Program Files/Microsoft/Edge/Application/msedge.exe'];
for (const x of p) console.log((fs.existsSync(x) ? 'FOUND ' : 'missing ') + x);
const pw = process.env.LOCALAPPDATA + '/ms-playwright';
console.log('playwright browsers: ' + (fs.existsSync(pw) ? fs.readdirSync(pw).join(',') : 'none'));
