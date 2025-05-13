# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:c0f429e16b13e583da7e5a6ec20dd656d325d88e6819cafe0adb0828976529dc
ARG TARGETARCH
WORKDIR /
COPY bin/manager-linux.${TARGETARCH} /manager
USER 65532:65532

ENTRYPOINT ["/manager"]
