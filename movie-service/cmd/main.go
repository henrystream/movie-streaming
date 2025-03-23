package main

import (
	"context"
	"log"
	"net"

	pb "github.com/henrystream/movie-streaming/movie-service/proto"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedMovieServiceServer
	movies []pb.MovieResponse
}

func (s *server) CreateMovie(ctx context.Context, req *pb.CreateMovieRequest) (*pb.MovieResponse, error) {
	log.Printf("CreateMovie: %v", req)
	movie := &pb.MovieResponse{
		Id:    int32(len(s.movies) + 1),
		Title: req.Title,
		Genre: req.Genre,
		Year:  req.Year,
	}
	s.movies = append(s.movies, *movie)
	return movie, nil
}

func (s *server) GetMovie(ctx context.Context, req *pb.GetMovieRequest) (*pb.MovieResponse, error) {
	for _, m := range s.movies {
		if m.Id == req.Id {
			return &m, nil
		}
	}
	return nil, grpc.Errorf(5, "Movie not found")
}

func (s *server) ListMovies(ctx context.Context, req *pb.EmptyRequest) (*pb.ListMoviesResponse, error) {
	rm := []*pb.MovieResponse{}
	for _, r := range s.movies {
		rm = append(rm, &r)
	}
	return &pb.ListMoviesResponse{Movies: rm}, nil
}

func (s *server) UpdateMovie(ctx context.Context, req *pb.UpdateMovieRequest) (*pb.MovieResponse, error) {
	for i, m := range s.movies {
		if m.Id == req.Id {
			s.movies[i] = pb.MovieResponse{
				Id:    req.Id,
				Title: req.Title,
				Genre: req.Genre,
				Year:  req.Year,
			}
			return &s.movies[i], nil
		}
	}
	return nil, grpc.Errorf(5, "Movie not found")
}

func (s *server) DeleteMovie(ctx context.Context, req *pb.DeleteMovieRequest) (*pb.DeleteMovieResponse, error) {
	for i, m := range s.movies {
		if m.Id == req.Id {
			s.movies = append(s.movies[:i], s.movies[i+1:]...)
			return &pb.DeleteMovieResponse{Success: true}, nil
		}
	}
	return &pb.DeleteMovieResponse{Success: false}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterMovieServiceServer(s, &server{})
	log.Println("Movie Service on :8082")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
