#!/usr/bin/env bash
# Better safe then sorry
set -e -o pipefail
if [[ $MAKEFLAGS == *"--debug"* ]]; then
  set -x
fi

BASE_DIR=$(git rev-parse --show-toplevel)
GENERATED_DIR="${BASE_DIR}"/hack/.generated
mkdir -p $GENERATED_DIR
if ! "${BASE_DIR}"/hack/jwt-expired.sh; then
  JWT_EXPIRED=true
else
  JWT_EXPIRED=false
fi



if [[ "$JWT_EXPIRED" = true ]]; then
  jf rt access-token-create --expiry 7776000 --refreshable > "${GENERATED_DIR}"/artifactory-access-token.json
  cat < "${GENERATED_DIR}"/artifactory-access-token.json | jq -r .access_token > "${GENERATED_DIR}"/artifactory-bearer-token.json;
  echo $("${BASE_DIR}"/hack/userid.sh) > "${GENERATED_DIR}"/artifactory-user;
fi
