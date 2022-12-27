{
  'targets': [{
    'target_name': 'binding',
    'actions': [{
      'action_name': 'gobuild',
      'outputs': ['<(INTERMEDIATE_DIR)/golib<(STATIC_LIB_SUFFIX)'],
      'inputs': [
        'gobuild.py',
        'go.mod',
        '<!@(go list -f \'{{ range .GoFiles }}{{ $.Dir }}/{{ . }} {{ end }}{{ range .CgoFiles }}{{ $.Dir }}/{{ . }} {{ end }}\' ./...)',
      ],
      'action': ['python3', 'gobuild.py', '<@(_outputs)', '>(_defines)', '>(_include_dirs)'],
    }],
    'conditions': [
      # TODO: Other platforms
      ['OS=="linux"', {
        'ldflags+': ['-Wl,--whole-archive,<(INTERMEDIATE_DIR)/golib<(STATIC_LIB_SUFFIX),--no-whole-archive'],
      }],
      ['OS=="mac"', {
        'xcode_settings': {
          'OTHER_LDFLAGS+': ['-Wl,-force_load,<(INTERMEDIATE_DIR)/golib<(STATIC_LIB_SUFFIX)'],
        },
      }],
    ],
  }],
}
