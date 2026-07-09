# Multi-stage build: this Dockerfile has two stages — "build" and the final stage.
#
# Stage 1 (build): uses a full Go toolchain image to compile the binary. The Go
# compiler, source code, and all build tools live here — but none of them end up
# in the final image.
#
# Stage 2 (final): starts from a minimal base image and copies in only the
# compiled binary and the page templates. The Go toolchain is left behind.
# Result: a much smaller, more secure image with no build-time dependencies
# present at runtime.
#
# Static game assets (script.js, CSS, models, etc.) are served exclusively by
# nginx from its own document root. The app has no file server of its own.
#
# Docker Hardened Images (DHI): both base images are pulled from dhi.io, Docker's
# registry of hardened images. dhi.io/golang:1.25-alpine-dev is the hardened Go
# toolchain; dhi.io/static is a hardened, near-empty image for running statically
# compiled binaries. Compared to official Docker Hub images, DHI images are built
# from a minimal base, have known-good package versions, and are regularly
# updated to reduce the attack surface of the containers you ship.
#
# Build context: the repo root is used (not app/) so this Dockerfile can COPY
# templates/ into the image alongside the Go binary.

FROM dhi.io/golang:1.25-alpine-dev AS build
WORKDIR /src
COPY app/ .
RUN CGO_ENABLED=0 go build -mod=vendor -o /out/app .
# Create an empty /repo directory so the final image has a mount point for
# the host .git directory. dhi.io/static has no shell, so we make it here.
RUN mkdir /out/repo

# The final image contains only the compiled binary and the frontend assets.
# No Go toolchain, no source code, no build cache — just what the app needs to run.
FROM dhi.io/static:20260611-alpine3.24
COPY --from=build /out/app /app
COPY --from=build /out/repo /repo
COPY templates/ /templates
ENTRYPOINT ["/app"]
