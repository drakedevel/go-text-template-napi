{
  'targets': [{
    'target_name': '<(module_name)',
    'type': 'none',
    'actions': [{
      'action_name': 'gobuild',
      'defines': [
        'NAPI_VERSION=<(napi_build_version)',
      ],
      'outputs': ['<(PRODUCT_DIR)/<(module_name).node'],
      'inputs': [
        'build-helpers/gobuild.py',
        'go.mod',
        '<!@(node build-helpers/listfiles.js)',
      ],
      'action': ['python3', 'build-helpers/gobuild.py', '<@(_outputs)', '>(_defines)', '>(_include_dirs)'],
      'conditions': [
        ['OS=="win"', {'inputs+': ['<(PRODUCT_DIR)/node_api.a']}],
      ],
    }],
    'conditions': [
      ['OS=="win"', {
        'actions': [{
          'action_name': 'gen_node_api',
          'outputs': ['<(PRODUCT_DIR)/node_api.a'],
          'inputs': ['build-helpers/gen_node_api_def.js'],
          'action': ['node', 'build-helpers/gen_node_api_def.js', '<@(_outputs)'],
        }],
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
