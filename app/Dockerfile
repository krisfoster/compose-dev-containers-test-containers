FROM dhi.io/golang:1.25-alpine-dev AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/app .

FROM dhi.io/static:20260611-alpine3.24
COPY --from=build /out/app /app
ENTRYPOINT ["/app"]
