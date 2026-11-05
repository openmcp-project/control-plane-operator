# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3
ARG TARGETARCH
WORKDIR /
COPY bin/manager-linux.${TARGETARCH} /manager
USER 65532:65532

ENTRYPOINT ["/manager"]
