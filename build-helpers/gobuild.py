import os
import shlex
import subprocess
import sys


def main():
    if len(sys.argv) != 4:
        print("Usage: gobuild.py <output> <defines> <includedirs>")
        sys.exit(1)
    out_path = sys.argv[1]
    defines = shlex.split(sys.argv[2])
    include_dirs = shlex.split(sys.argv[3])
    cflags = [f'-D{d}' for d in defines] + [f'-I{i}' for i in include_dirs]
    if sys.platform.startswith('darwin'):
        cflags.append('-mmacosx-version-min=10.13')
    ldflags = []
    if os.name == 'nt':
        ldflags.append(os.path.join(os.path.dirname(out_path), 'node_api.a'))
    buildflags = ['-buildmode=c-shared', '-o', out_path]
    if os.environ.get('GO_TEXT_TEMPLATE_NAPI_COVERAGE') == 'true':
        buildflags.extend(['-cover', '-tags=coverage'])
    if os.environ.get('CI') == 'true':
        buildflags.append('-ldflags=-s -w')
    subprocess.run(
        ['go', 'build'] + buildflags + ['.'],
        check=True,
        cwd=os.path.join(os.path.dirname(__file__), '..'),
        env=dict(os.environ, CGO_CFLAGS=shlex.join(cflags),
                 CGO_LDFLAGS=shlex.join(ldflags)),
    )


if __name__ == '__main__':
    main()
