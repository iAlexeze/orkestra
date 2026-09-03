FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETARCH
COPY ork-${TARGETARCH} /usr/local/bin/ork

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/ork"]
