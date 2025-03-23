FROM golang:1.24.1 AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY proto/user-service ./proto/user-service/
COPY proto/movie-service ./proto/movie-service/
RUN go mod download
COPY cmd ./cmd/
RUN CGO_ENABLED=0 GOOS=linux go build -o rest-gateway ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/rest-gateway .
EXPOSE 8080
CMD ["./rest-gateway"]