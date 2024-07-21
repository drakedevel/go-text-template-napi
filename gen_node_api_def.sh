#!/bin/bash
set -eu -o pipefail
gendef - "$(node -p process.execPath)" | grep -E "^(LIBRARY|EXPORTS|napi_|node_api_)" > node_api.def
dlltool --input-def node_api.def --output-delaylib node_api.a
