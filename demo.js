const {Template} = require('.');

const tmpl = new Template("test");
console.log(tmpl);
console.log(tmpl.name());

tmpl.parse("dot: {{ . }}")
console.log(tmpl.execute('hello world'))

console.log(tmpl.execute(['hello', 'world']))
console.log(tmpl.execute({'hello': 'world', [42]: 'stuff'}))

const tmpl2 = new Template("test2");
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

const tmpl3 = new Template("test3");
tmpl3.delims("<<", ">>");
tmpl3.parse("foo: << .foo >>");
console.log(tmpl3.execute({foo: 'foo'}));

const tmpl4 = tmpl3.new('test4');
tmpl4.parse('<< template "test3" . >> / bar: << .bar >>')
console.log(tmpl4.execute({foo: 'foo', bar: 'bar'}));

const tmpl5 = new Template("test5");
tmpl5.funcs({foo: () => {console.log('in func'); return {key: 'success!'};}});
tmpl5.parse('{{ foo }}{{ foo "bar" 123 3.5 9223372036854775807 true }}');
console.log(tmpl5.execute({}));

const tmpl6a = new Template("test6a");
const tmpl6b = new Template("test6b");
tmpl6a.funcs({foo: () => {console.log('in foo'); return `foo: ${tmpl6b.execute({})}`;}});
tmpl6b.funcs({bar: () => {console.log('in bar'); return `bar`;}});
tmpl6a.parse(`6a: {{ foo }}`);
tmpl6b.parse(`6b: {{ bar }}`);
console.log(tmpl6a.execute({}));
console.log(tmpl6a.definedTemplates(), tmpl6a.templates().map(t => t.name()))

const tmpl7 = new Template("test7")
tmpl7.funcs({f(...args) { return `in JS: ${JSON.stringify(args)}`; }})
tmpl7.parse('{{ f . }}')
for (const value of [0, 0.5, NaN, true, {a: 123, b: "cde"}, [], "str", null]) {
  console.log(value, tmpl7.execute(value));
}
