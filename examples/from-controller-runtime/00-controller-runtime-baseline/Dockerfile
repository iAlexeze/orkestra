FROM golang:1.22 AS builder
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY api/ api/
COPY controller/ controller/
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o manager ./main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]
