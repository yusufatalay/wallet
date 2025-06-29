package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yusufatalay/wallet/assets/internal/database"
	"github.com/yusufatalay/wallet/assets/internal/models"
	pb "github.com/yusufatalay/wallet/proto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// server holds grpc dependencies.
type Server struct {
	pb.UnimplementedAssetServiceServer
	Client       *mongo.Client
	AssetsStore  database.AssetStore
	TxStore      database.TransactionStore
	WalletClient pb.WalletServiceClient
}

// Deposit , deposits an asset to a wallet.
func (s *Server) Deposit(ctx context.Context, req *pb.DepositRequest) (*pb.TransactionResponse, error) {
	ownerRes, err := s.WalletClient.ValidateWalletOwner(ctx, &pb.ValidateWalletOwnerRequest{
		WalletId: req.WalletId,
	})
	if err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "Could not authenticate wallet owner: %v", err)
	}
	if !ownerRes.IsOwner {
		return nil, status.Error(codes.PermissionDenied, "User has no permission for this wallet")
	}

	walletID, _ := primitive.ObjectIDFromHex(req.WalletId)
	filter := bson.M{"walletId": walletID, "name": req.AssetName}
	update := bson.M{"$inc": bson.M{"amount": req.Amount}}
	_, err = s.AssetsStore.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update balance: %v", err)
	}

	newTx := models.Transaction{
		ID:        primitive.NewObjectID(),
		WalletID:  walletID,
		Type:      "DEPOSIT",
		AssetName: req.AssetName,
		Amount:    req.Amount,
		Status:    "COMPLETED",
		Timestamp: time.Now(),
	}
	res, err := s.TxStore.InsertOne(ctx, &newTx)
	if err != nil {
		return &pb.TransactionResponse{
			TransactionId: "LOGGING_ERROR",
			Status:        "COMPLETED",
		}, nil
	}

	return &pb.TransactionResponse{
		TransactionId: res.InsertedID.(primitive.ObjectID).Hex(),
		Status:        "COMPLETED",
	}, nil
}

func (s *Server) Withdraw(ctx context.Context, req *pb.WithdrawRequest) (*pb.TransactionResponse, error) {
	ownerRes, err := s.WalletClient.ValidateWalletOwner(ctx, &pb.ValidateWalletOwnerRequest{
		WalletId: req.WalletId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "could not authenticate wallet owner")
	}
	if !ownerRes.IsOwner {
		return nil, status.Error(codes.PermissionDenied, "user has no permission for this wallet")
	}

	walletID, err := primitive.ObjectIDFromHex(req.WalletId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid walletId format")
	}

	var asset *models.Asset
	filter := bson.M{"walletId": walletID, "name": req.AssetName}
	asset, err = s.AssetsStore.FindOne(ctx, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, status.Errorf(codes.FailedPrecondition, "insufficient funds")
		}

		return nil, status.Errorf(codes.Internal, "could not check balance: %v", err)
	}

	if asset.Amount < req.Amount {
		return nil, status.Errorf(codes.FailedPrecondition, "insufficient funds. balance: %f, asked for: %f",
			asset.Amount, req.Amount)
	}

	update := bson.M{"$inc": bson.M{"amount": -req.Amount}}
	_, err = s.AssetsStore.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update balance: %v", err)
	}

	newTx := models.Transaction{
		ID:        primitive.NewObjectID(),
		WalletID:  walletID,
		Type:      "WITHDRAW",
		AssetName: req.AssetName,
		Amount:    req.Amount,
		Status:    "COMPLETED",
		Timestamp: time.Now(),
	}
	res, err := s.TxStore.InsertOne(ctx, &newTx)
	if err != nil {
		log.Printf("failed to create transaction log (WITHDRAW): %v", err)

		return &pb.TransactionResponse{TransactionId: "LOGGING_ERROR", Status: "COMPLETED",
			Message: "Withdraw successful, logging failed."}, nil
	}

	return &pb.TransactionResponse{TransactionId: res.InsertedID.(primitive.ObjectID).Hex(),
		Status: "COMPLETED"}, nil
}

func (s *Server) GetWalletAssets(ctx context.Context, req *pb.GetWalletAssetsRequest) (*pb.GetWalletAssetsResponse,
	error) {
	ownerRes, err := s.WalletClient.ValidateWalletOwner(ctx, &pb.ValidateWalletOwnerRequest{WalletId: req.WalletId})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not auhthenticate wallet owner: %v", err)
	}

	if !ownerRes.IsOwner {
		return nil, status.Error(codes.PermissionDenied, "user has no permission for this wallet")
	}

	walletID, err := primitive.ObjectIDFromHex(req.WalletId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid walletId format")
	}

	dbAssets, err := s.AssetsStore.FindAssetsByWalletID(ctx, walletID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Varlıklar alınamadı: %v", err)
	}

	var protoAssets []*pb.Asset
	for _, asset := range dbAssets {
		protoAssets = append(protoAssets, &pb.Asset{
			Name:   asset.Name,
			Amount: asset.Amount,
		})
	}

	return &pb.GetWalletAssetsResponse{Assets: protoAssets}, nil
}

func (s *Server) Transfer(ctx context.Context, req *pb.TransferRequest) (*pb.TransactionResponse, error) {
	ownerRes, err := s.WalletClient.ValidateWalletOwner(ctx, &pb.ValidateWalletOwnerRequest{
		WalletId: req.FromWalletId})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not auhthenticate wallet owner: %v", err)
	}

	if !ownerRes.IsOwner {
		return nil, status.Error(codes.PermissionDenied, "user has no permission for this wallet")
	}

	fromWalletID, err := primitive.ObjectIDFromHex(req.FromWalletId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid wallet_if format")
	}

	session, err := s.Client.StartSession()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to start db session: %v", err)
	}
	defer session.EndSession(ctx)

	var transactionResult *pb.TransactionResponse
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		var sourceAsset *models.Asset
		sourceFilter := bson.M{"walletId": fromWalletID, "name": req.AssetName}
		sourceAsset, err := s.AssetsStore.FindOne(sessCtx, sourceFilter)
		if err != nil || sourceAsset.Amount < req.Amount {
			return nil, fmt.Errorf("insufficient funds: %v ", err)
		}

		sourceUpdate := bson.M{"$inc": bson.M{"amount": -req.Amount}}
		_, err = s.AssetsStore.UpdateOne(sessCtx, sourceFilter, sourceUpdate)

		if err != nil {
			return nil, fmt.Errorf("failed to update wallet: %w", err)
		}

		destWallet, err := s.WalletClient.FindWalletByAddress(sessCtx, &pb.FindWalletByAddressRequest{
			Address: req.ToWalletAddress,
			Network: req.ToWalletNetwork,
		})
		if err != nil {
			return nil, fmt.Errorf("destination wallet not found via RPC: %w", err)
		}
		destWalletID, _ := primitive.ObjectIDFromHex(destWallet.WalletId)

		destAssetFilter := bson.M{"walletId": destWalletID, "name": req.AssetName}
		destUpdate := bson.M{"$inc": bson.M{"amount": req.Amount}}
		_, err = s.AssetsStore.UpdateOne(sessCtx, destAssetFilter, destUpdate, options.Update().SetUpsert(true))
		if err != nil {
			return nil, fmt.Errorf("failed to update destination wallet: %w", err)
		}

		newTx := models.Transaction{
			ID:        primitive.NewObjectID(),
			Type:      models.TypeTransfer,
			Status:    models.StatusCompleted,
			AssetName: req.AssetName,
			Amount:    req.Amount,
			TransferDetails: models.TransferDetails{
				FromWalletID:    fromWalletID,
				ToWalletAddress: req.ToWalletAddress,
				ToWalletNetwork: req.ToWalletNetwork,
			},
			Timestamp: time.Now(),
		}
		res, err := s.TxStore.InsertOne(sessCtx, &newTx)
		if err != nil {
			return nil, fmt.Errorf("failed to log transfer record: %w", err)
		}

		transactionResult = &pb.TransactionResponse{
			TransactionId: res.InsertedID.(primitive.ObjectID).Hex(),
			Status:        string(models.StatusCompleted),
		}

		return transactionResult, nil
	}

	_, err = session.WithTransaction(ctx, callback)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Transfer failed: %v", err)
	}

	return transactionResult, nil
}

func (s *Server) ScheduleDeposit(ctx context.Context, req *pb.ScheduleDepositRequest) (*pb.TransactionResponse, error) {
	ownerRes, err := s.WalletClient.ValidateWalletOwner(ctx, &pb.ValidateWalletOwnerRequest{WalletId: req.WalletId})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not auhthenticate wallet owner: %v", err)
	}

	if !ownerRes.IsOwner {
		return nil, status.Error(codes.PermissionDenied, "user has no permission for this wallet")
	}

	walletID, _ := primitive.ObjectIDFromHex(req.WalletId)
	newTx := models.Transaction{
		ID:           primitive.NewObjectID(),
		Type:         models.TypeDeposit,
		Status:       models.StatusScheduled,
		AssetName:    req.AssetName,
		Amount:       req.Amount,
		WalletID:     walletID,
		ScheduledFor: req.ScheduledFor.AsTime(),
		Timestamp:    time.Now(),
	}

	res, err := s.TxStore.InsertOne(ctx, &newTx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to schedule deposit: %v", err)
	}

	return &pb.TransactionResponse{TransactionId: res.InsertedID.(primitive.ObjectID).Hex(),
		Status: string(models.StatusScheduled)}, nil
}

func (s *Server) ScheduleWithdraw(ctx context.Context, req *pb.ScheduleWithdrawRequest) (*pb.TransactionResponse,
	error) {
	ownerRes, err := s.WalletClient.ValidateWalletOwner(ctx, &pb.ValidateWalletOwnerRequest{WalletId: req.WalletId})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not auhthenticate wallet owner: %v", err)
	}

	if !ownerRes.IsOwner {
		return nil, status.Error(codes.PermissionDenied, "user has no permission for this wallet")
	}

	walletID, _ := primitive.ObjectIDFromHex(req.WalletId)

	var asset *models.Asset
	filter := bson.M{"walletId": walletID, "name": req.AssetName}
	asset, err = s.AssetsStore.FindOne(ctx, filter)
	if err != nil || asset.Amount < req.Amount {
		return nil, status.Errorf(codes.FailedPrecondition, "insufficient funds")
	}

	newTx := models.Transaction{
		ID:           primitive.NewObjectID(),
		Type:         models.TypeWithdraw,
		Status:       models.StatusScheduled,
		AssetName:    req.AssetName,
		Amount:       req.Amount,
		WalletID:     walletID,
		ScheduledFor: req.ScheduledFor.AsTime(),
		Timestamp:    time.Now(),
	}
	res, err := s.TxStore.InsertOne(ctx, &newTx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to schedule withdraw: %v", err)
	}

	return &pb.TransactionResponse{TransactionId: res.InsertedID.(primitive.ObjectID).Hex(),
		Status: string(models.StatusScheduled)}, nil
}

func (s *Server) ScheduleTransfer(ctx context.Context, req *pb.ScheduleTransferRequest) (*pb.TransactionResponse,
	error) {
	ownerRes, err := s.WalletClient.ValidateWalletOwner(ctx, &pb.ValidateWalletOwnerRequest{
		WalletId: req.FromWalletId})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not auhthenticate wallet owner: %v", err)
	}

	if !ownerRes.IsOwner {
		return nil, status.Error(codes.PermissionDenied, "user has no permission for this wallet")
	}

	fromWalletID, _ := primitive.ObjectIDFromHex(req.FromWalletId)
	var asset *models.Asset
	filter := bson.M{"walletId": fromWalletID, "name": req.AssetName}
	asset, err = s.AssetsStore.FindOne(ctx, filter)
	if err != nil || asset.Amount < req.Amount {
		return nil, status.Errorf(codes.FailedPrecondition, "insufficient funds")
	}

	newTx := models.Transaction{
		ID:        primitive.NewObjectID(),
		Type:      models.TypeTransfer,
		Status:    models.StatusScheduled,
		AssetName: req.AssetName,
		Amount:    req.Amount,
		TransferDetails: models.TransferDetails{FromWalletID: fromWalletID,
			ToWalletAddress: req.ToWalletAddress, ToWalletNetwork: req.ToWalletNetwork},
		ScheduledFor: req.ScheduledFor.AsTime(),
		Timestamp:    time.Now(),
	}
	res, err := s.TxStore.InsertOne(ctx, &newTx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to schedule transfer: %v", err)
	}

	return &pb.TransactionResponse{TransactionId: res.InsertedID.(primitive.ObjectID).Hex(),
		Status: string(models.StatusScheduled)}, nil
}

func (s *Server) ListTransactions(ctx context.Context, req *pb.ListTransactionsRequest) (*pb.ListTransactionsResponse,
	error) {
	filter := bson.M{}

	if req.WalletId != "" {
		ownerRes, err := s.WalletClient.ValidateWalletOwner(ctx, &pb.ValidateWalletOwnerRequest{
			WalletId: req.WalletId})

		if err != nil {
			return nil, status.Errorf(codes.Internal, "could not auhthenticate wallet owner: %v", err)
		}

		if !ownerRes.IsOwner {
			return nil, status.Error(codes.PermissionDenied, "user has no permission for this wallet")
		}

		walletID, _ := primitive.ObjectIDFromHex(req.WalletId)
		filter["$or"] = []bson.M{
			{"walletId": walletID},
			{"fromWalletId": walletID},
		}
	} else {
		return nil, status.Error(codes.InvalidArgument, "wallet_id is required")
	}

	if req.Status != "" {
		filter["status"] = req.Status
	}

	dbTransactions, err := s.TxStore.Find(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "İşlemler alınamadı: %v", err)
	}

	var protoTransactions []*pb.Transaction
	for _, tx := range dbTransactions {
		protoTransactions = append(protoTransactions, &pb.Transaction{
			Id:              tx.ID.Hex(),
			Type:            string(tx.Type),
			Status:          string(tx.Status),
			AssetName:       tx.AssetName,
			Amount:          tx.Amount,
			FromWalletId:    tx.FromWalletID.Hex(),
			ToWalletAddress: tx.ToWalletAddress,
			ToWalletNetwork: tx.ToWalletNetwork,
			WalletId:        tx.WalletID.Hex(),
			ScheduledFor:    timestamppb.New(tx.ScheduledFor),
			Timestamp:       timestamppb.New(tx.Timestamp),
			ProcessNotes:    tx.ProcessNotes,
		})
	}

	return &pb.ListTransactionsResponse{Transactions: protoTransactions}, nil
}
