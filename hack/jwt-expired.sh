#!/usr/bin/env bash
# Better safe then sorry
set -e -o pipefail
if [[ $MAKEFLAGS == *"--debug"* ]]; then
  set -x
fi
BASE_DIR=$(git rev-parse --show-toplevel)
TOKEN_FILE="$BASE_DIR"/hack/.generated/artifactory-access-token.json
if [[ ! -f $TOKEN_FILE ]]; then
  exit 1
fi

expiry=$(cat < "$TOKEN_FILE"  | jq  -r .access_token | jq -r -R 'split(".") | .[1]'  | jq -r -R '@base64d' | jq  -r .exp)

now=$(date '+%s')

# It's math time.

if [ $((expiry - now)) -gt 3600 ]; then
  exit 0
fi

exit 1
