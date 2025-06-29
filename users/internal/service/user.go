package service

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v4"
	pb "github.com/yusufatalay/wallet/proto"
	"github.com/yusufatalay/wallet/users/internal/database"
	"github.com/yusufatalay/wallet/users/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	pb.UnimplementedUserServiceServer
	Store     database.UserStore
	JwtSecret string
}

// Register, registers a new user.
func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "Email and password fields are required")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to hash the password: %v", err)
	}

	newUser := models.User{
		ID:       primitive.NewObjectID(),
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	_, err = s.Store.InsertOne(ctx, &newUser)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, status.Error(codes.AlreadyExists, "Email already registered")
		}

		return nil, status.Errorf(codes.Internal, "Could not register user: %v", err)
	}

	return &pb.RegisterResponse{UserId: newUser.ID.Hex()}, nil
}

// Login, logs the user in and returns JWT.
func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := s.Store.FindOneByEmail(ctx, req.Email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Error(codes.NotFound, "User not found with given e-mail")
		}

		return nil, status.Errorf(codes.Internal, "Failed to search for user: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "e-mail or password incorrect")
	}

	claims := jwt.MapClaims{
		"user_id": user.ID.Hex(),
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.JwtSecret))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not create a token: %v", err)
	}

	return &pb.LoginResponse{AccessToken: tokenString}, nil
}
