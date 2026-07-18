# Backend Dockerfile

FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY ./go.mod ./go.sum ./
RUN go mod download
# Now copy the rest of the source code
COPY . .
# Stamp the build with the git short SHA. Passed in from docker-compose / CI
# (the .git dir isn't copied into the image, so it must come in as a build arg).
# Defaults to "dev" for a plain `docker build` with no arg.
ARG GIT_SHA=dev
RUN go build -ldflags "-X main.version=${GIT_SHA}" -o server .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server ./server
EXPOSE 8080
CMD ["./server"]
