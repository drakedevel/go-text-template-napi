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
        'gobuild.py',
        'go.mod',
        '<!@(bash listfiles.sh)',
      ],
      'action': ['python3', 'gobuild.py', '<@(_outputs)', '>(_defines)', '>(_include_dirs)'],
    }],
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
