# Multi-stage build: this Dockerfile has two stages — "build" and the final stage.
#
# Stage 1 (build): uses a full Go toolchain image to compile the binary. The Go
# compiler, source code, and all build tools live here — but none of them end up
# in the final image.
#
# Stage 2 (final): starts from a minimal base image and copies in only the
# compiled binary and the frontend assets. The Go toolchain is left behind.
# Result: a much smaller, more secure image with no build-time dependencies
# present at runtime.
#
# Docker Hardened Images (DHI): both base images are pulled from dhi.io, Docker's
# registry of hardened images. dhi.io/golang:1.25-alpine-dev is the hardened Go
# toolchain; dhi.io/static is a hardened, near-empty image for running statically
# compiled binaries. Compared to official Docker Hub images, DHI images are built
# from a minimal base, have known-good package versions, and are regularly
# updated to reduce the attack surface of the containers you ship.
#
# Build context: the repo root is used (not app/) so this Dockerfile can COPY
# frontend/game into the image alongside the Go binary.

FROM dhi.io/golang:1.25-alpine-dev AS build
WORKDIR /src
COPY app/go.mod app/go.sum ./
RUN go mod download
COPY app/ .
RUN CGO_ENABLED=0 go build -o /out/app .

# The final image contains only the compiled binary and the frontend assets.
# No Go toolchain, no source code, no build cache — just what the app needs to run.
FROM dhi.io/static:20260611-alpine3.24
COPY --from=build /out/app /app
COPY frontend/game /frontend
COPY container-obstacles.gif /frontend/container-obstacles.gif
ENTRYPOINT ["/app"]
