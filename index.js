const binding = require('./build/Release/binding');
console.log(binding);

const tmpl = new binding.Template("name");
console.log(tmpl);
tmpl.foo();
tmpl.foo();
tmpl.bar();
tmpl.bar();
