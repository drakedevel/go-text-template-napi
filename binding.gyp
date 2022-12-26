{
  'targets': [{
    'target_name': 'binding',
    'actions': [{
      'action_name': 'gobuild',
      'outputs': ['<(INTERMEDIATE_DIR)/golib<(STATIC_LIB_SUFFIX)'],
      'inputs': [
        'gobuild.py',
        'go.mod',
        'main.go',
        'exports.go',
      ],
      'action': ['python', 'gobuild.py', '<@(_outputs)', '>(_defines)', '>(_include_dirs)'],
    }],
    'conditions': [
      # TODO: Other platforms
      ['OS=="linux"', {
        'ldflags+': [
          '-Wl,--whole-archive',
          '<(INTERMEDIATE_DIR)/golib<(STATIC_LIB_SUFFIX)',
          '-Wl,--no-whole-archive'
        ],
      }],
    ],
  }],
}
