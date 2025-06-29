package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/yusufatalay/wallet/proto"
	"github.com/yusufatalay/wallet/users/internal/database"
	"github.com/yusufatalay/wallet/users/internal/service"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

const (
	grpcPort        = ":50051"
	dbName          = "wallet_db"
	usersCollection = "users"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI env is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	mongoClient, err := database.ConnectMongo(ctx, mongoURI)
	if err != nil {
		log.Fatalf("MongoDB failed to connect: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}()
	log.Println("MongoDB connection successful.")

	usersColl := mongoClient.Database(dbName).Collection(usersCollection)
	indexModel := mongo.IndexModel{
		Keys:    bson.M{"email": 1},
		Options: options.Index().SetUnique(true),
	}
	_, err = usersColl.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Fatalf("Failed to create unique index for email: %v", err)
	}

	userStore := database.NewUserStore(usersColl)

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Could not listen port : %v", err)
	}

	s := grpc.NewServer()
	userServer := &service.Server{
		Store:     userStore,
		JwtSecret: jwtSecret,
	}
	pb.RegisterUserServiceServer(s, userServer)

	log.Printf("gRPC server starting at port %s ...", grpcPort)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to start the server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")
	s.GracefulStop()
	log.Println("Server shut down.")
}
