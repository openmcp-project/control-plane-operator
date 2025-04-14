#!/bin/bash

set -euo pipefail
source "$(realpath "$(dirname $0)/environment.sh")"

echo
echo "> Building binaries ..."
(
  cd "$PROJECT_ROOT"
  for comp in ${COMPONENTS//,/ }; do
    for pf in ${PLATFORMS//,/ }; do
      echo "> Building binary for component '$comp' ($pf) ..." | indent 1
      os=${pf%/*}
      arch=${pf#*/}
      CGO_ENABLED=0 GOFIPS140=v1.0.0 GOOS=$os GOARCH=$arch go build -a -o bin/${comp}-${os}.${arch} cmd/main.go | indent 2
    done
  done
)
