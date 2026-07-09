# Multi-stage build for the commits microservice.
# Build context: commits-service/ directory.
#
# Stage 1 (build): compiles the Go binary using the hardened Go toolchain image.
# Stage 2 (final): minimal dhi.io/static image with only the binary and an empty
# /repo mount point (the .git volume is mounted at runtime by docker-compose).
#
# Docker Hardened Images (DHI): both base images are from dhi.io, the same
# registry used by the app service.

FROM dhi.io/golang:1.25-alpine-dev AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/commits-service .
# Create an empty /repo directory so the final image has a mount point for
# the host .git directory.
RUN mkdir /out/repo

FROM dhi.io/static:20260611-alpine3.24
COPY --from=build /out/commits-service /commits-service
COPY --from=build /out/repo /repo
ENTRYPOINT ["/commits-service"]
