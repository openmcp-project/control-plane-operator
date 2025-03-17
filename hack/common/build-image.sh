#!/bin/bash

set -euo pipefail
source "$(realpath "$(dirname $0)/environment.sh")"

if [[ -z ${IMAGE_REGISTRY:-} ]]; then
	IMAGE_REGISTRY=$("$COMMON_SCRIPT_DIR/get-registry.sh" -i)
fi

VERSION=$("$COMMON_SCRIPT_DIR/get-version.sh")

DOCKER_BUILDER_NAME="mcp-multiarch-builder"
if ! docker buildx ls | grep "$DOCKER_BUILDER_NAME" >/dev/null; then
	docker buildx create --name "$DOCKER_BUILDER_NAME"
fi

# remove temporary Dockerfile on exit
trap "rm -f \"${PROJECT_ROOT}/Dockerfile.tmp\"" EXIT

echo
echo "> Building images ..."
for comp in ${COMPONENTS//,/ }; do
	for pf in ${PLATFORMS//,/ }; do
		os=${pf%/*}
		arch=${pf#*/}
		img="${IMAGE_REGISTRY}/${comp}:${VERSION}-${os}-${arch}"
		echo "> Building image for component '$comp' ($pf): $img ..." | indent 1
		cat "${COMMON_SCRIPT_DIR}/Dockerfile" | sed "s/<component>/$comp/g" > "${PROJECT_ROOT}/Dockerfile.tmp"
		docker buildx build --builder ${DOCKER_BUILDER_NAME} --load --build-arg COMPONENT=${comp} --platform ${pf} -t $img -f Dockerfile.tmp "${PROJECT_ROOT}" | indent 2
	done
done

docker buildx rm "$DOCKER_BUILDER_NAME"