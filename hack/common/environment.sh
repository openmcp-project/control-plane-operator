#!/bin/bash

export COMMON_SCRIPT_DIR="$(realpath "$(dirname ${BASH_SOURCE[0]})")"
source "$COMMON_SCRIPT_DIR/lib.sh"
export PROJECT_ROOT="${PROJECT_ROOT:-$(realpath "$COMMON_SCRIPT_DIR/../..")}"
export COMPONENT_DEFINITION_FILE="${COMPONENT_DEFINITION_FILE:-"$PROJECT_ROOT/components/components.yaml"}"

export LOCALBIN="${LOCALBIN:-"$PROJECT_ROOT/bin"}"
export HELM="${HELM:-"$LOCALBIN/helm"}"
export JQ="${JQ:-"$LOCALBIN/jq"}"
export FORMATTER=${FORMATTER:-"$LOCALBIN/goimports"}
export OCM="${OCM:-"$LOCALBIN/ocm"}"
export YAML2JSON="${YAML2JSON:-"$LOCALBIN/yaml2json"}"

if [[ -f "$COMMON_SCRIPT_DIR/../environment.sh" ]]; then
  source "$COMMON_SCRIPT_DIR/../environment.sh"
fi
