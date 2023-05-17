This library provides JavaScript bindings to [Go's text/template
package][text-template] package via [Node-API][node-api]. Nearly the full API is
supported, including custom template functions written in JavaScript.

For example, this JavaScript program:
```ts
import {Template} from 'go-text-template-napi';

const template = new Template("name")
  .funcs({double: l => [...l, ...l]})
  .parse(`{{ range double .targets }}Hello, {{ . }}!\n{{ end }}`);
process.stdout.write(template.executeString({targets: ['user', 'world']}));
```

is equivalent to this Go program:
```go
package main

import (
        "os"
        "text/template"
)

func double(list []any) []any {
        return append(list, list...)
}

func main() {
        tmpl := template.New("name").Funcs(template.FuncMap{"double": double})
        template.Must(tmpl.Parse("{{ range double .targets }}Hello, {{ . }}!\n{{ end }}"))
        err := tmpl.Execute(os.Stdout, map[string]any{"targets": []any{"user", "world"}})
        if err != nil {
                panic(err)
        }
}
```

Both output:
```text
Hello, user!
Hello, world!
Hello, user!
Hello, world!
```

**WARNING**: Do **not** use this library with untrusted templates, and use
_extreme_ caution when passing untrusted data to trusted templates. See the
[Warnings](#warnings) section for more details.

[node-api]: https://nodejs.org/api/node-api.html
[text-template]: https://pkg.go.dev/text/template

### Experimental Sprig Support
[Sprig][sprig] template functions can be enabled by calling the `addSprigFuncs`
method on `Template`. This API is subject to change.

[sprig]: https://github.com/Masterminds/sprig

### Requirements
The native component requires Node-API version 8, which is available on all
supported release lines of Node. It's tested on Linux and MacOS, and will
probably work on any Unix-like operating system. Windows support is possible but
currently absent. PRs to expand platform support are welcome!

Pre-built binaries are available for Linux and MacOS on 64-bit x86 and ARM. If
they are unavailable for your platform, the install script will try to build
them automatically, which requires Go 1.18 or later.

### Warnings
Importing this package will spin up a Go runtime _within_ your Node
process. Should this library have a bug that results in a Go panic, it will take
the entire process with it.

Additionally, be aware that Go's memory usage is not subject to Node's heap size
limits. Be sure that adequate additional memory is available if your workload
causes significant Go memory usage.

This library currently buffers the full template output with no size limit, so
an untrusted template can trivially DoS your application by generating an output
larger than your available memory.

### API Limitations
A few less-useful parts of the API are unimplemented:
- The `parse` subpackage and related functions (they're documented as internal
  interfaces)
- All functions operating on a `fs.FS` virtual fileystem
- The `*Escape` helper functions that write to a `io.Writer`

Additionally, the `Execute` and `ExecuteTemplate` methods return strings instead
of taking a `Writer` parameter. This is faster than a streaming interface given
the FFI overhead, but it also makes it impossible to limit the memory usage of
untrusted templates (per the Warnings section). Support for the streaming
interface is planned for future releases.
