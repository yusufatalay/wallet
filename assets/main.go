package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/yusufatalay/wallet/assets/internal/database"
	"github.com/yusufatalay/wallet/assets/internal/scheduler"
	"github.com/yusufatalay/wallet/assets/internal/service"
	"github.com/yusufatalay/wallet/pkg/interceptors"
	"github.com/yusufatalay/wallet/pkg/middleware"
	pb "github.com/yusufatalay/wallet/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	grpcPort              = ":50053"
	dbName                = "wallet_db"
	assetsCollectionName  = "assets"
	txCollectionName      = "transactions"
	walletsCollectionName = "wallets"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	walletsAddr := os.Getenv("WALLETS_SERVICE_ADDR")
	jwtSecret := os.Getenv("JWT_SECRET")

	mongoClient, err := database.ConnectMongo(ctx, mongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to mongodb: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}()

	conn, err := grpc.NewClient(walletsAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptors.ClientAuthInterceptor()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to wallets service: %v", err)
	}
	defer conn.Close()
	walletClient := pb.NewWalletServiceClient(conn)

	lis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen to port %s: %v", grpcPort, err)
	}

	assetStore := database.NewAssetStore(mongoClient.Database(dbName).Collection(assetsCollectionName))
	txStore := database.NewTransactionStore(mongoClient.Database(dbName).Collection(txCollectionName))

	s := grpc.NewServer(grpc.UnaryInterceptor(middleware.AuthInterceptor(jwtSecret)))

	assetServer := &service.Server{
		Client:       mongoClient,
		AssetsStore:  assetStore,
		TxStore:      txStore,
		WalletClient: walletClient,
	}
	pb.RegisterAssetServiceServer(s, assetServer)

	worker := &scheduler.Worker{
		Client:       mongoClient,
		Assets:       mongoClient.Database(dbName).Collection(assetsCollectionName),
		Transactions: mongoClient.Database(dbName).Collection(txCollectionName),
		WalletClient: walletClient,
	}
	go worker.Start(ctx)

	log.Printf("Asset grpc server starting at %s", grpcPort)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to start the server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down the server...")
	s.GracefulStop()
	log.Println("Server gracefully stopped.")
}
