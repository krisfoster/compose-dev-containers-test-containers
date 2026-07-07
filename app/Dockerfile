# Build context: repo root (required to COPY frontend/game alongside the Go source).
# This Dockerfile is used in both Docker Compose and Kubernetes deployments.
# The resulting image is self-contained: the Go binary and all frontend game assets
# are embedded, so no volume mounts are needed at runtime.

FROM dhi.io/golang:1.25-alpine-dev AS build
WORKDIR /src
COPY app/go.mod app/go.sum ./
RUN go mod download
COPY app/ .
RUN CGO_ENABLED=0 go build -o /out/app .

FROM dhi.io/static:20260611-alpine3.24
COPY --from=build /out/app /app
COPY frontend/game /frontend
ENTRYPOINT ["/app"]
