package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"time"

	"github.com/henrystream/movie-streaming/movie-service/db"
	"github.com/henrystream/movie-streaming/movie-service/proto"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var dbPool *pgxpool.Pool
var queries *db.Queries
var kafkaWriter *kafka.Writer
var redisClient *redis.Client

func initDB() {
	var err error
	connStr := "postgres://postgres:password@postgres:5432/movies?sslmode=disable"
	for i := 0; i < 10; i++ {
		log.Printf("Attempt %d: Connecting to DB at %s", i+1, connStr)
		dbPool, err = pgxpool.New(context.Background(), connStr)
		if err != nil {
			log.Printf("Attempt %d: Failed to create DB pool: %v, retrying in 2s...", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}
		err = dbPool.Ping(context.Background())
		if err == nil {
			log.Println("DB connection successful")
			break
		}
		log.Printf("Attempt %d: Failed to ping DB: %v, retrying in 2s...", i+1, err)
		dbPool.Close()
		dbPool = nil
		time.Sleep(2 * time.Second)
	}
	if dbPool == nil {
		log.Fatalf("Failed to establish DB connection after 10 attempts")
	}
	if err != nil {
		log.Fatalf("Last connection attempt failed with: %v", err)
	}

	log.Println("Applying schema...")
	schema, err := os.ReadFile("./db/schema.sql")
	if err != nil {
		log.Fatalf("Failed to read schema.sql: %v", err)
	}
	_, err = dbPool.Exec(context.Background(), string(schema))
	if err != nil {
		log.Fatalf("Failed to apply schema: %v", err)
	}

	queries = db.New(dbPool)
	log.Println("DB initialized successfully")
}

func initKafka() {
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP("kafka:9092"),
		Topic:    "movie-events",
		Balancer: &kafka.LeastBytes{},
	}
	for i := 0; i < 10; i++ {
		log.Printf("Attempt %d: Connecting to Kafka at kafka:9092", i+1)
		conn, err := kafka.Dial("tcp", "kafka:9092")
		if err != nil {
			log.Printf("Attempt %d: Failed to connect to Kafka: %v, retrying in 2s...", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}
		err = conn.CreateTopics(kafka.TopicConfig{
			Topic:             "movie-events",
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
		if err != nil && err.Error() != "topic already exists" {
			log.Printf("Attempt %d: Failed to create topic: %v, retrying in 2s...", i+1, err)
			conn.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		conn.Close()
		log.Println("Kafka connection and topic setup successful")
		return
	}
	log.Fatalf("Failed to connect to Kafka or create topic after retries")
}

func initRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	_, err := redisClient.Ping(context.Background()).Result()
	for i := 0; i < 10 && err != nil; i++ {
		log.Printf("Attempt %d: Failed to connect to Redis: %v, retrying in 2s...", i+1, err)
		time.Sleep(2 * time.Second)
		_, err = redisClient.Ping(context.Background()).Result()
	}
	if err != nil {
		log.Fatalf("Failed to connect to Redis after retries: %v", err)
	}
	log.Println("Redis connection successful")
}

type movieServer struct {
	proto.UnimplementedMovieServiceServer
}

func (s *movieServer) CreateMovie(ctx context.Context, req *proto.CreateMovieRequest) (*proto.MovieResponse, error) {
	movie, err := queries.CreateMovie(ctx, db.CreateMovieParams{
		Title: req.Title,
		Genre: req.Genre,
		Year:  req.Year,
	})
	if err != nil {
		return nil, err
	}
	resp := &proto.MovieResponse{
		Id:    int32(movie.ID),
		Title: movie.Title,
		Genre: movie.Genre,
		Year:  movie.Year,
	}
	// Cache in Redis
	cacheData, _ := json.Marshal(resp)
	redisClient.Set(ctx, "movie:"+string(rune(movie.ID)), cacheData, 24*time.Hour)
	// Publish Kafka event
	go func() {
		err = kafkaWriter.WriteMessages(context.Background(), kafka.Message{
			Key:   []byte("create_movie"),
			Value: cacheData,
		})
		if err != nil {
			log.Printf("Kafka write failed: %v", err)
		}
	}()
	return resp, nil
}

func (s *movieServer) GetMovie(ctx context.Context, req *proto.GetMovieRequest) (*proto.MovieResponse, error) {
	// Check Redis cache
	cacheKey := "movie:" + string(rune(req.Id))
	cached, err := redisClient.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var resp proto.MovieResponse
		json.Unmarshal(cached, &resp)
		return &resp, nil
	}
	// Cache miss, query DB
	movie, err := queries.GetMovie(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	resp := &proto.MovieResponse{
		Id:    int32(movie.ID),
		Title: movie.Title,
		Genre: movie.Genre,
		Year:  movie.Year,
	}
	// Update cache
	cacheData, _ := json.Marshal(resp)
	redisClient.Set(ctx, cacheKey, cacheData, 24*time.Hour)
	return resp, nil
}

func (s *movieServer) UpdateMovie(ctx context.Context, req *proto.UpdateMovieRequest) (*proto.MovieResponse, error) {
	movie, err := queries.UpdateMovie(ctx, db.UpdateMovieParams{
		ID:    req.Id,
		Title: req.Title,
		Genre: req.Genre,
		Year:  req.Year,
	})
	if err != nil {
		return nil, err
	}
	resp := &proto.MovieResponse{
		Id:    int32(movie.ID),
		Title: movie.Title,
		Genre: movie.Genre,
		Year:  movie.Year,
	}
	// Update cache
	cacheData, _ := json.Marshal(resp)
	redisClient.Set(ctx, "movie:"+string(rune(movie.ID)), cacheData, 24*time.Hour)
	// Publish Kafka event
	go func() {
		err = kafkaWriter.WriteMessages(context.Background(), kafka.Message{
			Key:   []byte("update_movie"),
			Value: cacheData,
		})
		if err != nil {
			log.Printf("Kafka write failed: %v", err)
		}
	}()
	return resp, nil
}

func (s *movieServer) DeleteMovie(ctx context.Context, req *proto.DeleteMovieRequest) (*proto.DeleteMovieResponse, error) {
	_, err := queries.DeleteMovie(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	// Clear cache
	redisClient.Del(ctx, "movie:"+string(rune(req.Id)))
	// Publish Kafka event
	go func() {
		err = kafkaWriter.WriteMessages(context.Background(), kafka.Message{
			Key:   []byte("delete_movie"),
			Value: []byte(string(rune(req.Id))),
		})
		if err != nil {
			log.Printf("Kafka write failed: %v", err)
		}
	}()
	return &proto.DeleteMovieResponse{}, nil
}

func (s *movieServer) ListMovies(ctx context.Context, req *proto.EmptyRequest) (*proto.ListMoviesResponse, error) {
	movies, err := queries.ListMovies(ctx)
	if err != nil {
		return nil, err
	}
	resp := &proto.ListMoviesResponse{}
	for _, m := range movies {
		resp.Movies = append(resp.Movies, &proto.MovieResponse{
			Id:    int32(m.ID),
			Title: m.Title,
			Genre: m.Genre,
			Year:  m.Year,
		})
	}
	return resp, nil
}

func main() {
	initDB()
	defer dbPool.Close()
	initKafka()
	initRedis()

	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterMovieServiceServer(s, &movieServer{})
	reflection.Register(s)
	log.Println("Movie service on :8082")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
