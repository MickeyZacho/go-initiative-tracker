# Backend Dockerfile

FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY ./go.mod ./go.sum ./
RUN go mod download
# Now copy the rest of the source code
COPY . .
RUN go build -o server .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server ./server
EXPOSE 8080
CMD ["./server"]
