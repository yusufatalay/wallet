package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yusufatalay/wallet/pkg/middleware"
	pb "github.com/yusufatalay/wallet/proto"
	"github.com/yusufatalay/wallet/wallets/internal/database"
	"github.com/yusufatalay/wallet/wallets/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server, holds gRPC service dependencies.
type Server struct {
	pb.UnimplementedWalletServiceServer
	Store database.WalletStore
	Cache database.CacheStore
}

// CreateWallet , creates a new wallet.
func (s *Server) CreateWallet(ctx context.Context, req *pb.CreateWalletRequest) (*pb.CreateWalletResponse, error) {
	userIDHex, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "Could not found UserID")
	}

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid UserID format")
	}

	newWallet := models.Wallet{
		ID:      primitive.NewObjectID(),
		UserID:  userID,
		Address: req.Address,
		Network: req.Network,
	}

	_, err = s.Store.InsertOne(ctx, &newWallet)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not create wallet: %v", err)
	}

	cacheKey := fmt.Sprintf("user:%s:wallets", userIDHex)
	s.Cache.Del(ctx, cacheKey)

	return &pb.CreateWalletResponse{
		Wallet: &pb.Wallet{
			WalletId: newWallet.ID.Hex(),
			UserId:   newWallet.UserID.Hex(),
			Address:  newWallet.Address,
			Network:  newWallet.Network,
		},
	}, nil
}

// GetWallets, returns user's wallets from cache.
func (s *Server) GetWallets(ctx context.Context, _ *pb.GetWalletsRequest) (*pb.GetWalletsResponse, error) {
	userIDHex, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "Could not found UserID")
	}

	cacheKey := fmt.Sprintf("user:%s:wallets", userIDHex)

	// check cache
	cachedWallets, err := s.Cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var wallets []*pb.Wallet
		if json.Unmarshal([]byte(cachedWallets), &wallets) == nil {
			return &pb.GetWalletsResponse{Wallets: wallets}, nil
		}
	}

	// cache miss, check db
	userID, _ := primitive.ObjectIDFromHex(userIDHex)
	dbWallets, err := s.Store.FindWalletsByUserID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get wallets: %v", err)
	}

	var protoWallets []*pb.Wallet
	for _, wallet := range dbWallets {
		protoWallets = append(protoWallets, &pb.Wallet{
			WalletId: wallet.ID.Hex(),
			UserId:   wallet.UserID.Hex(),
			Address:  wallet.Address,
			Network:  wallet.Network,
		})
	}
	// cache result
	walletsJSON, _ := json.Marshal(protoWallets)
	s.Cache.Set(ctx, cacheKey, walletsJSON, time.Minute*5)

	return &pb.GetWalletsResponse{Wallets: protoWallets}, nil
}

// ValidateWalletOwner (internal) checks if a user owns a spesific wallet.
func (s *Server) ValidateWalletOwner(ctx context.Context, req *pb.ValidateWalletOwnerRequest) (
	*pb.ValidateWalletOwnerResponse, error) {
	// get userID from context
	userIDHex, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok || userIDHex == "" {
		return nil, status.Error(codes.Unauthenticated, "missing or faulty token")
	}

	walletID, err := primitive.ObjectIDFromHex(req.WalletId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid walletId format")
	}

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid userID format")
	}

	filter := bson.M{
		"_id":    walletID,
		"userId": userID,
	}

	count, err := s.Store.CountDocuments(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error when authenticating wallet owner: %v", err)
	}

	return &pb.ValidateWalletOwnerResponse{IsOwner: count > 0}, nil
}

// FindWalletByAddress returns wallet based on address and network.
func (s *Server) FindWalletByAddress(ctx context.Context, req *pb.FindWalletByAddressRequest) (*pb.Wallet, error) {
	filter := bson.M{"address": req.Address, "network": req.Network}

	wallet, err := s.Store.FindOne(ctx, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Errorf(codes.NotFound, "no wallet found with this address and network")
		}

		return nil, status.Errorf(codes.Internal, "error occurred while looking for wallet: %v", err)
	}

	return &pb.Wallet{
		WalletId: wallet.ID.Hex(),
		UserId:   wallet.UserID.Hex(),
		Address:  wallet.Address,
		Network:  wallet.Network,
	}, nil
}
