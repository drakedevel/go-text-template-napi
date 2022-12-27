const binding = require('./build/Release/binding');
console.log(binding);

const tmpl = new binding.Template("myname");
console.log(tmpl);
console.log(tmpl.name());

tmpl.parse("foo: {{ . }}")
console.log(tmpl.execute('hello world'))
