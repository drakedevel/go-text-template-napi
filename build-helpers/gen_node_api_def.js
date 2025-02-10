const { execFileSync } = require('child_process');
const fs = require('fs');

const outFile = process.argv[2];
const defFile = `${outFile}.def`;
const dlltool = process.env.DLLTOOL || 'dlltool';

const defStr = execFileSync('gendef', ['-', process.execPath], {
  encoding: 'utf8',
  maxBuffer: 8 * 1024 * 1024,
});
const defLines = defStr
  .trimEnd()
  .split('\n')
  .filter((l) => /^(LIBRARY|EXPORTS|napi_|node_api_)/.test(l));
fs.writeFileSync(`${process.argv[2]}.def`, `${defLines.join('\n')}\n`);
execFileSync(dlltool, ['--input-def', defFile, '--output-delaylib', outFile]);
