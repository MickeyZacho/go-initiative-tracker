# syntax=docker/dockerfile:1
# Backend Dockerfile

FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY ./go.mod ./go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
# Now copy the rest of the source code
COPY . .
# Stamp the build with the git short SHA. Passed in from docker-compose / CI
# (the .git dir isn't copied into the image, so it must come in as a build arg).
# Defaults to "dev" for a plain `docker build` with no arg.
ARG GIT_SHA=dev
# Cache mounts persist Go's module + build caches across image builds. GIT_SHA
# changes every commit (correctly invalidating this layer), but with the caches
# a new build only recompiles changed packages and relinks — not the whole tree.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags "-X main.version=${GIT_SHA}" -o server .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server ./server
EXPOSE 8080
CMD ["./server"]
