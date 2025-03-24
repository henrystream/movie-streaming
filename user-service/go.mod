module github.com/henrystream/movie-streaming/user-service

go 1.24.1

require (
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/henrystream/movie-streaming/user-service/proto v1.0.0
	github.com/jackc/pgx/v5 v5.7.3
	github.com/segmentio/kafka-go v0.4.47
	golang.org/x/crypto v0.32.0
	google.golang.org/grpc v1.71.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/protobuf v1.36.5 // indirect
)

replace github.com/henrystream/movie-streaming/user-service/proto v1.0.0 => ./proto
