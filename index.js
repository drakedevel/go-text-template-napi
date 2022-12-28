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
console.log(tmpl2.execute({
  'hello': 'world',
   [42]: 'stuff',
   nullKey: null,
   undefKey: undefined,
   num1: 42,
   num2: 1.5,
   num3: 9007199254740993n,
   num4: 18446744073709551618n,
   num5: -18446744073709551618n,
   //sym: Symbol('sym')
}));

const tmpl3 = new binding.Template("test3");
tmpl3.delims("<<", ">>");
tmpl3.parse("foo: << .foo >>");
console.log(tmpl3.execute({foo: 'foo'}));

const tmpl4 = tmpl3.new('test4');
tmpl4.parse('<< template "test3" . >> / bar: << .bar >>')
console.log(tmpl4.execute({foo: 'foo', bar: 'bar'}));
