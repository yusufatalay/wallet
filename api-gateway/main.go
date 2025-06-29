package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/yusufatalay/wallet/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func passThroughHeaderMatcher(key string) (string, bool) {
	if strings.ToLower(key) == "authorization" {
		return key, true
	}

	return runtime.DefaultHeaderMatcher(key)
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// main mux will serve for both grpc gateway and static files
	mainMux := http.NewServeMux()

	gwMux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(passThroughHeaderMatcher),
	)

	usersAddr := os.Getenv("USERS_SERVICE_ADDR")
	if usersAddr == "" {
		log.Fatal("FATAL: USERS_SERVICE_ADDR env variable is not set")
	}
	log.Printf("INFO: User service address: %s", usersAddr)

	usersOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err := pb.RegisterUserServiceHandlerFromEndpoint(ctx, gwMux, usersAddr, usersOpts)
	if err != nil {
		log.Fatalf("Could not connect to Users service: %v", err)
	}
	log.Println("INFO: http handler for users service is set.")

	walletsAddr := os.Getenv("WALLETS_SERVICE_ADDR")
	if walletsAddr == "" {
		log.Fatal("FATAL: WALLETS_SERVICE_ADDR env variable is not set.")
	}
	log.Printf("INFO: Wallets service address: %s", walletsAddr)

	walletsOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = pb.RegisterWalletServiceHandlerFromEndpoint(ctx, gwMux, walletsAddr, walletsOpts)
	if err != nil {
		log.Fatalf("Could not connect to Wallets service: %v", err)
	}
	log.Println("INFO: http handler for wallets service is set.")

	assetsAddr := os.Getenv("ASSETS_SERVICE_ADDR")
	if assetsAddr == "" {
		log.Fatal("FATAL: ASSETS_SERVICE_ADDR env variable is not set.")
	}
	log.Printf("INFO: Assets service address: %s", assetsAddr)

	assetsOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	err = pb.RegisterAssetServiceHandlerFromEndpoint(ctx, gwMux, assetsAddr, assetsOpts)
	if err != nil {
		log.Fatalf("Could not connect to Assets service: %v", err)
	}
	log.Println("INFO: http handler for assets service is set.")

	mainMux.Handle("/", gwMux)

	// serve swagger ui
	fs := http.FileServer(http.Dir("./proto"))
	mainMux.Handle("/swagger/", http.StripPrefix("/swagger/", fs))

	staticFS := http.FileServer(http.Dir("./static"))
	mainMux.Handle("/docs/", http.StripPrefix("/docs/", staticFS))

	log.Println("API Gateway starting at port :8080 ...")
	log.Println("Document http://localhost:8080/docs/")
	if err := http.ListenAndServe(":8080", mainMux); err != nil {
		log.Fatalf("Failed to start Gateway server: %v", err)
	}
}
