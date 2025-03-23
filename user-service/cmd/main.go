package main

import (
	"context"
	"log"
	"net"

	pb "github.com/henrystream/movie-streaming/user-service/proto"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedUserServiceServer
}

func (s *server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	log.Printf("Register: %v", req)
	return &pb.RegisterResponse{
		Id:             1, // Dummy ID
		Email:          req.Email,
		MembershipType: req.MembershipType,
	}, nil
}

func (s *server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	log.Printf("Login: %v", req)
	return &pb.LoginResponse{Token: "dummy-token"}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, &server{})
	log.Println("User Service on :8081")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
