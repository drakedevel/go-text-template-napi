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
    buildflags = ['-buildmode=c-archive', '-o', out_path]
    if os.environ.get('CI') == 'true':
        buildflags.append('-ldflags=-s -w')
    subprocess.run(
        ['go', 'build'] + buildflags + ['.'],
        check=True,
        env=dict(os.environ, CGO_CFLAGS=shlex.join(cflags)),
    )


if __name__ == '__main__':
    main()
