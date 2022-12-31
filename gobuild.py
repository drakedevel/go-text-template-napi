import os
import shlex
import subprocess
import sys


def main():
    if len(sys.argv) != 5:
        print("Usage: gobuild.py <output> <defines> <includedirs> <libs>")
        sys.exit(1)
    out_path = sys.argv[1]
    defines = shlex.split(sys.argv[2])
    include_dirs = shlex.split(sys.argv[3])
    libs = shlex.split(sys.argv[4])
    cflags = [f'-D{d}' for d in defines] + [f'-I{i}' for i in include_dirs]
    if sys.platform.startswith('darwin'):
        cflags.append('-mmacosx-version-min=10.13')
    ldflags = []
    if os.name == 'nt':
        node_lib = [lib for lib in libs if 'node.lib' in lib][0]
        node_lib = node_lib.replace('-l', '')
        ldflags.extend([f'-L{os.path.dirname(node_lib)}', '-lnode'])
    buildflags = ['-buildmode=c-shared', '-o', out_path]
    if os.environ.get('GO_TEXT_TEMPLATE_NAPI_COVERAGE') == 'true':
        buildflags.extend(['-cover', '-tags=coverage'])
    #if os.environ.get('CI') == 'true':
    #    buildflags.append('-ldflags=-s -w')
    subprocess.run(
        ['go', 'build'] + buildflags + ['.'],
        check=True,
        cwd=os.path.dirname(__file__),
        env=dict(os.environ, CGO_CFLAGS=shlex.join(cflags),
                 CGO_LDFLAGS=shlex.join(ldflags)),
    )


if __name__ == '__main__':
    main()
