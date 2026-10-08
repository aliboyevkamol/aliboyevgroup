const fs = require('fs');
const pages = ['index','about','skills','projects','project-detail','store','product','blog','article','contact','cart','checkout','payment','login','register','profile'];
let problems = 0;
for (const f of pages) {
  const h = fs.readFileSync(f + '.html', 'utf8');
  const m = h.match(/<body data-page="([^"]+)"/);
  const links = [...h.matchAll(/href="([^"#][^"]*\.html)[^"]*"/g)].map(x => x[1]);
  const miss = [...new Set(links.filter(l => !fs.existsSync(l)))];
  const ids = [...h.matchAll(/id="([^"]+)"/g)].map(x => x[1]);
  const dup = [...new Set(ids.filter((v, i) => ids.indexOf(v) !== i))];
  const css = /style\.css/.test(h), js = /<script src="app\.js"/.test(h);
  const bad = [];
  if (!m) bad.push('no data-page'); if (!css) bad.push('no css'); if (!js) bad.push('no app.js');
  if (miss.length) bad.push('broken links: ' + miss.join(','));
  if (dup.length) bad.push('duplicate ids: ' + dup.join(','));
  if (bad.length) problems++;
  console.log(`${(bad.length ? 'FAIL' : 'PASS')} | ${f.padEnd(15)} page=${(m ? m[1] : '-').padEnd(15)} ${bad.join('; ')}`);
}
console.log(`\n${problems === 0 ? 'ALL PAGE-STRUCTURE CHECKS PASSED' : problems + ' pages with problems'}`);
