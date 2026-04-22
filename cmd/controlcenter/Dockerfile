FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETARCH
COPY orkcc-${TARGETARCH} /usr/local/bin/orkcc

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/orkcc"]
