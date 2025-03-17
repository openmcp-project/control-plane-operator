#!/bin/bash -eu

set -euo pipefail

if [[ -z ${BASE_REGISTRY:-} ]]; then
  BASE_REGISTRY=ghcr.io/openmcp-project
fi

if [[ -z ${IMAGE_REGISTRY:-} ]]; then
  IMAGE_REGISTRY=$BASE_REGISTRY
fi
if [[ -z ${CHART_REGISTRY:-} ]]; then
  CHART_REGISTRY=$BASE_REGISTRY/charts
fi
if [[ -z ${COMPONENT_REGISTRY:-} ]]; then
  COMPONENT_REGISTRY=$BASE_REGISTRY/components
fi

mode="BASE_"

while [[ "$#" -gt 0 ]]; do
  case ${1:-} in
    "-i"|"--image")
      mode="IMAGE_"
      ;;
    "-h"|"--helm")
      mode="CHART_"
      ;;
    "-c"|"--component")
      mode="COMPONENT_"
      ;;
    *)
      echo "invalid argument: $1" 1>&2
      exit 1
      ;;
  esac
  shift
done

eval echo "\$${mode}REGISTRY"
