FROM golang:1.24.1 AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY proto ./proto/
COPY db ./db/
RUN go mod download
COPY cmd ./cmd/
RUN CGO_ENABLED=0 GOOS=linux go build -o movie-service ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/movie-service .
COPY --from=builder /app/db/schema.sql ./db/
EXPOSE 8082
CMD ["./movie-service"]
