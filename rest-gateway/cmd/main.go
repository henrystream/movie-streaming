package main

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	pb_movie "github.com/henrystream/movie-streaming/movie-service/proto"
	pb_user "github.com/henrystream/movie-streaming/user-service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RestGateway struct {
	userClient  pb_user.UserServiceClient
	movieClient pb_movie.MovieServiceClient
}

func NewRestGateway() *RestGateway {
	userConn, err := grpc.Dial("user-service:8081", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to user-service: %v", err)
	}
	movieConn, err := grpc.Dial("movie-service:8082", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to movie-service: %v", err)
	}
	return &RestGateway{
		userClient:  pb_user.NewUserServiceClient(userConn),
		movieClient: pb_movie.NewMovieServiceClient(movieConn),
	}
}

func (g *RestGateway) RegisterRoutes(r *gin.Engine) {
	r.POST("/users/register", g.RegisterUser)
	r.POST("/users/login", g.LoginUser)
	r.POST("/movies", g.CreateMovie)
	r.GET("/movies/:id", g.GetMovie)
	r.GET("/movies", g.ListMovies)
	r.PUT("/movies/:id", g.UpdateMovie)
	r.DELETE("/movies/:id", g.DeleteMovie)
	r.GET("/health", g.Healthcheck)

}

func (g *RestGateway) Healthcheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "healthy"})
}

func (g *RestGateway) RegisterUser(c *gin.Context) {
	var req pb_user.RegisterRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := g.userClient.Register(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (g *RestGateway) LoginUser(c *gin.Context) {
	var req pb_user.LoginRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := g.userClient.Login(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (g *RestGateway) CreateMovie(c *gin.Context) {
	var req pb_movie.CreateMovieRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := g.movieClient.CreateMovie(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (g *RestGateway) GetMovie(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	resp, err := g.movieClient.GetMovie(context.Background(), &pb_movie.GetMovieRequest{Id: int32(id)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (g *RestGateway) ListMovies(c *gin.Context) {
	resp, err := g.movieClient.ListMovies(context.Background(), &pb_movie.EmptyRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp.Movies)
}

func (g *RestGateway) UpdateMovie(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var req pb_movie.UpdateMovieRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Id = int32(id)
	resp, err := g.movieClient.UpdateMovie(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (g *RestGateway) DeleteMovie(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	resp, err := g.movieClient.DeleteMovie(context.Background(), &pb_movie.DeleteMovieRequest{Id: int32(id)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func main() {
	r := gin.Default()
	gateway := NewRestGateway()
	gateway.RegisterRoutes(r)

	log.Println("REST API Gateway on :8080")
	r.Run(":8080")
}
