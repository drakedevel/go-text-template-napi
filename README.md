This library provides JavaScript bindings to [Go's text/template
package][text-template] package via [N-API][n-api]. Nearly the full API is
supported, including custom template functions written in JavaScript.

**Do not use this library with untrusted templates**, and use caution when
passing untrusted data to trusted templates. See [Warnings](#warnings) section
for more details.

[n-api]: https://nodejs.org/api/n-api.html
[text-template]: https://pkg.go.dev/text/template

### Requirements
The native component requires N-API version 8, which is available on all
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
Several parts of the API are unimplemented:
- The `parse` subpackage and related functions (they're documented as internal
  interfaces)
- All functions operating on a `fs.FS` virtual fileystem
- The `*Escape` helper functions (support is planned)

Additionally, the `Execute` and `ExecuteTemplate` methods return strings instead
of taking a `Writer` parameter. This is faster than a streaming interface given
the FFI overhead, but it also makes it impossible to limit the memory usage of
untrusted templates (per the Warnings section). Support for the streaming
interface is planned for future releases.
