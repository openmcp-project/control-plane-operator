#!/usr/bin/env bash
# Better safe then sorry
set -e -o pipefail
if [[ $MAKEFLAGS == *"--debug"* ]]; then
  set -x
fi
BASE_DIR=$(git rev-parse --show-toplevel)
TOKEN_FILE="$BASE_DIR"/hack/.generated/artifactory-access-token.json
cat < "$TOKEN_FILE"  | jq  -r .access_token | jq -r -R 'split(".") | .[1]'  | jq -r -R '@base64d' | jq  -r .sub | awk -F/ '{print $(NF-0)}'
