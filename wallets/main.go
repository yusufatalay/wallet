package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/yusufatalay/wallet/pkg/middleware"
	pb "github.com/yusufatalay/wallet/proto"
	"github.com/yusufatalay/wallet/wallets/internal/cache"
	"github.com/yusufatalay/wallet/wallets/internal/database"
	"github.com/yusufatalay/wallet/wallets/internal/service"
	"google.golang.org/grpc"
)

const (
	grpcPort          = ":50052"
	dbName            = "wallet_db"
	walletsCollection = "wallets"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	redisAddr := os.Getenv("REDIS_ADDR")
	jwtSecret := os.Getenv("JWT_SECRET")

	mongoClient, err := database.ConnectMongo(ctx, mongoURI)
	if err != nil {
		log.Fatalf("Could not connect to mongodb: %v", err)
	}
	redisClient, err := cache.ConnectRedis(ctx, redisAddr)
	if err != nil {
		log.Fatalf("Could not connect to redis: %v", err)
	}

	log.Println("Database and Cache connections are successful.")

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Could not listen the port: %v", err)
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.AuthInterceptor(jwtSecret)),
	)

	walletsColl := mongoClient.Database(dbName).Collection(walletsCollection)
	walletStore := database.NewMongoStore(walletsColl)
	walletServer := &service.Server{
		Store: walletStore,
		Cache: redisClient,
	}
	pb.RegisterWalletServiceServer(s, walletServer)

	log.Printf("Wallets gRPC server starting at port %s ...", grpcPort)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Could not start the server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")
	s.GracefulStop()
	log.Println("Server shut down.")
}
