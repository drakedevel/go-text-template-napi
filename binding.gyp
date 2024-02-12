{
  'targets': [{
    'target_name': '<(module_name)',
    'actions': [{
      'action_name': 'gobuild',
      'defines': [
        'NAPI_VERSION=<(napi_build_version)',
      ],
      'outputs': ['<(INTERMEDIATE_DIR)/golib<(STATIC_LIB_SUFFIX)'],
      'inputs': [
        'gobuild.py',
        'go.mod',
        '<!@(go list -f \'{{ range .GoFiles }}{{ $.Dir }}/{{ . }} {{ end }}{{ range .CgoFiles }}{{ $.Dir }}/{{ . }} {{ end }}\' ./...)',
      ],
      'action': ['python3', 'gobuild.py', '<@(_outputs)', '>(_defines)', '>(_include_dirs)'],
    }],
    'conditions': [
      # TODO: Windows support
      ['OS in "aix freebsd linux netbsd openbsd solaris".split()', {
        'ldflags+': [
          # Linker flags used by Go 1.22 with -buildmode=c-shared
          # nodelete is especially important: since there is no way to stop the
          # Go runtime, it is never safe to unload a Go shared library. See
          # golang/go#11100 for more context.
          # TODO: Consider using c-shared mode directly
          '-Wl,-z,relro',
          '-Wl,-z,nodelete',
          '-Wl,-Bsymbolic',
          '-Wl,--whole-archive,<(INTERMEDIATE_DIR)/golib<(STATIC_LIB_SUFFIX),--no-whole-archive',
        ],
      }],
      ['OS=="mac"', {
        'xcode_settings': {
          'OTHER_LDFLAGS+': ['-Wl,-force_load,<(INTERMEDIATE_DIR)/golib<(STATIC_LIB_SUFFIX)'],
        },
      }],
    ],
  }, {
    'target_name': 'copy_build',
    'type': 'none',
    'dependencies': ['<(module_name)'],
    'copies': [{
      'files': [
        '<(PRODUCT_DIR)/<(module_name).node',
        'LICENSE',
        'NOTICE',
        'packaging/LICENSE.golang',
        'packaging/NOTICE.third-party',
      ],
      'destination': '<(module_path)'
    }],
    'conditions': [
      ['<!(test -d packaging/third-party && echo 1 || echo 0)==1', {
        'copies+': [{
          'files': ['packaging/third-party'],
          'destination': '<(module_path)',
        }],
      }],
    ],
  }],
}
