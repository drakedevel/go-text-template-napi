const binding = require('./build/Release/binding');
console.log(binding);

const tmpl = new binding.Template("test");
console.log(tmpl);
console.log(tmpl.name());

tmpl.parse("dot: {{ . }}")
console.log(tmpl.execute('hello world'))

console.log(tmpl.execute(['hello', 'world']))
console.log(tmpl.execute({'hello': 'world', [42]: 'stuff'}))

const tmpl2 = new binding.Template("test2");
tmpl2.parse('{{ range $k, $v := . }}k = {{ $k }}, v = {{ $v }}\n{{ end }}')
console.log(tmpl2.execute({'hello': 'world', [42]: 'stuff', nullKey: null, undefKey: undefined}))
