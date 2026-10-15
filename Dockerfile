FROM golang:1.27.1 AS build

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN --mount=type=cache,target=/root/.cache go mod download

# Copy the go source
COPY main.go main.go

# Build
RUN --mount=type=cache,target=/root/.cache \
    CGO_ENABLED=0 go build -o echo-server main.go

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /workspace/echo-server /echo-server
CMD ["/echo-server"]
