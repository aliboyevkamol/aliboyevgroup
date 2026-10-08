const d = require('./database/seeds/content.json');
console.log('top keys:', Object.keys(d));
for (const k of Object.keys(d)) {
  if (Array.isArray(d[k])) {
    console.log('---', k, 'len=', d[k].length);
    console.log(JSON.stringify(d[k][0]).slice(0, 400));
  }
}
