package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/henrystream/movie-streaming/user-service/db"
	"github.com/henrystream/movie-streaming/user-service/proto"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const jwtSecret = "your-secret-key" //In production, use env var or secret manager

var (
	dbPool      *pgxpool.Pool
	queries     *db.Queries
	kafkaWriter *kafka.Writer
)

type userServer struct {
	proto.UnimplementedUserServiceServer
}

func initDB() {
	var err error
	//connStr := "postgres://postgres:password@postgres:5433/users?sslmode=disable"
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_DB"))

	for i := 0; i < 10; i++ {
		log.Printf("Attempt %d: Connecting to DB at %s", i+1, connStr)
		dbPool, err = pgxpool.New(context.Background(), connStr)
		if err != nil {
			log.Printf("Attempt %d: Failed to create DB pool: %v, retrying in 2s...", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}
		log.Printf("Attempt %d: Pinging DB...", i+1)
		err = dbPool.Ping(context.Background())
		if err == nil {
			log.Println("DB connection successful")
			break
		}
		log.Printf("Attempt %d: Failed to ping DB: %v, retrying in 2s...", i+1, err)
		dbPool.Close()
		dbPool = nil // Explicitly reset to nil
		time.Sleep(2 * time.Second)
	}
	if dbPool == nil {
		log.Fatalf("Failed to establish DB connection after 10 attempts")
	}
	if err != nil {
		log.Fatalf("Last connection attempt failed with: %v", err)
	}

	// Apply schema
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
		Addr:     kafka.TCP(os.Getenv("KAFKA_BROKER")),
		Topic:    "user-events",
		Balancer: &kafka.LeastBytes{},
	}
	for i := 0; i < 10; i++ {
		log.Printf("Attempt %d: Connecting to Kafka at kafka:9092", i+1)
		conn, err := kafka.Dial("tcp", os.Getenv("KAFKA_BROKER"))
		if err != nil {
			log.Printf("Attempt %d: Failed to connect to Kafka: %v, retrying in 2s...", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}
		// Create topic if it doesn’t exist
		err = conn.CreateTopics(kafka.TopicConfig{
			Topic:             "user-events",
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
		if err != nil && err.Error() != "topic already exists" { // Ignore if topic exists
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

func (s *userServer) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user, err := queries.RegisterUser(ctx, db.RegisterUserParams{
		Email:          req.Email,
		Password:       string(hashed),
		MembershipType: req.MembershipType,
	})
	if err != nil {
		return nil, err
	}

	err = kafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte("register"),
		Value: []byte(req.Email),
	})
	if err != nil {
		log.Printf("Kafka write failed: %v", err)
	}

	return &proto.RegisterResponse{
		Id:             int32(user.ID),
		Email:          user.Email,
		MembershipType: user.MembershipType,
	}, nil
}

func (s *userServer) Login(ctx context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {
	user, err := queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err //could be user not found but keep it generic for now
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, err //Invalid password
	}

	//Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":    user.ID,
		"email": user.Email,
		"exp":   time.Now().Add(time.Hour * 24).Unix(), //24-hour expiry
	})
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, err
	}

	//Publish login event to kafka
	err = kafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte("login"),
		Value: []byte(req.Email),
	})
	if err != nil {
		log.Printf("Kafka write failed: %v", err)
	}

	return &proto.LoginResponse{Token: tokenString}, nil
}

func main() {
	initDB()
	defer dbPool.Close()
	initKafka()

	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterUserServiceServer(s, &userServer{})
	reflection.Register(s) // Enable reflection
	log.Println("User service on: 8081")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
