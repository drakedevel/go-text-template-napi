const { execFileSync } = require('child_process');
const path = require('path');

const mods = execFileSync('go', ['list', './...'], { encoding: 'utf8' });
for (const mod of mods.trimEnd().split('\n')) {
  const infoStr = execFileSync('go', ['list', '-json', mod], {
    encoding: 'utf8',
  });
  const info = JSON.parse(infoStr);
  for (const key of ['CXXFiles', 'CgoFiles', 'GoFiles']) {
    for (const file of info[key] ?? []) {
      console.log(path.join(info.Dir, file).replaceAll('\\', '\\\\'));
    }
  }
}
