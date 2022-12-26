{
  'targets': [{
    'target_name': 'binding',
    'actions': [{
      'action_name': 'gobuild',
      'outputs': ['<(INTERMEDIATE_DIR)/golib.a'],
      'inputs': [
        'go.mod',
        'main.go',
        'exports.go',
      ],
      'action': ['go', 'build', '-buildmode=c-archive', '-o', '<@(_outputs)', '.'],
    }],
    'conditions': [
      # TODO: Other platforms
      ['OS=="linux"', {
        'ldflags+': [
          '-Wl,--whole-archive',
          '<(INTERMEDIATE_DIR)/golib.a',
          '-Wl,--no-whole-archive'
        ],
      }],
    ],
  }],
}
