#!/bin/sh

set -o errexit
set -o nounset
set -o pipefail

# build example modules
cargo build --target wasm32-wasi

# download kubewarden modules
mkdir -p kubewarden/
(
	cd kubewarden/

	# download safe-annotations module
	if [ ! -f safe-annotations_v0.2.0.wasm ]; then
		curl -L -o safe-annotations_v0.2.0.wasm https://github.com/kubewarden/safe-annotations-policy/releases/download/v0.2.0/policy.wasm
	fi
)
