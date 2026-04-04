# Backend Dockerfile

FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY ./go.mod ./go.sum ./
RUN GOPROXY="https://goproxy.io" go mod download
# Now copy the rest of the source code
COPY . .
RUN go build -o server ./main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server ./server
COPY --from=builder /app/static ./static
# COPY --from=builder /app/templates ./templates
COPY --from=builder /app/initiative-tracker-db.session.sql ./initiative-tracker-db.session.sql
EXPOSE 8080
CMD ["./server"]
