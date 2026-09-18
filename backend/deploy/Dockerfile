# Single image containing every defShows service binary (auth, web, notifier,
# worker). Pick one at runtime via the command, e.g.
#   docker run <image> web
# Build with: docker build -f deploy/Dockerfile ..   (context is the backend/ dir)
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# -o /out/ builds each ./cmd/<name> main package into /out/<name>.
RUN CGO_ENABLED=0 go build -trimpath -o /out/ ./cmd/...

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
COPY --from=build /out/ /usr/local/bin/
USER app
# Default to the web API; override with `command:` (compose) or a run argument
# to launch auth / notifier / worker from the same image.
CMD ["web"]
