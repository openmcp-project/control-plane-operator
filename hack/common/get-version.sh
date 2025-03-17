#!/bin/bash -eu

if [[ -n ${EFFECTIVE_VERSION:-} ]] ; then
  # running in the pipeline use the provided EFFECTIVE_VERSION
  echo "$EFFECTIVE_VERSION"
  exit 0
fi

set -euo pipefail
source "$(realpath "$(dirname $0)/environment.sh")"

VERSION="$(cat "${PROJECT_ROOT}/VERSION")"

(
  cd "$PROJECT_ROOT"

  if [[ "$VERSION" = *-dev ]] ; then
    VERSION="$VERSION-$(git rev-parse HEAD)"
  fi
  
  echo "$VERSION"
)
