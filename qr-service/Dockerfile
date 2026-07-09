# Multi-stage build for the qr microservice.
# Build context: qr-service/ directory.
#
# Stage 1 (build): compiles the Go binary using the hardened Go toolchain image.
# Stage 2 (final): minimal dhi.io/static image with only the binary.
#
# Docker Hardened Images (DHI): both base images are from dhi.io, the same
# registry used by the app, commits-service, and scores-service.

FROM dhi.io/golang:1.25-alpine-dev AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/qr-service .

FROM dhi.io/static:20260611-alpine3.24
COPY --from=build /out/qr-service /qr-service
ENTRYPOINT ["/qr-service"]
