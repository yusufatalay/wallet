package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yusufatalay/wallet/assets/internal/models"
	"github.com/yusufatalay/wallet/pkg/middleware"
	pb "github.com/yusufatalay/wallet/proto"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

type MockAssetStore struct{ mock.Mock }

func (m *MockAssetStore) FindOne(ctx context.Context, filter interface{}) (*models.Asset, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*models.Asset), args.Error(1)
}

func (m *MockAssetStore) UpdateOne(ctx context.Context, f, u interface{},
	_ ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, f, u)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

func (m *MockAssetStore) FindAssetsByWalletID(ctx context.Context, walletID primitive.ObjectID) ([]models.Asset,
	error) {
	args := m.Called(ctx, walletID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]models.Asset), args.Error(1)
}

type MockTxStore struct{ mock.Mock }

func (m *MockTxStore) InsertOne(ctx context.Context, tx *models.Transaction) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, tx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*mongo.InsertOneResult), args.Error(1)
}

func (m *MockTxStore) Find(ctx context.Context, filter interface{}) ([]models.Transaction, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]models.Transaction), args.Error(1)
}

type MockWalletClient struct{ mock.Mock }

func (m *MockWalletClient) CreateWallet(_ context.Context, _ *pb.CreateWalletRequest,
	_ ...grpc.CallOption) (*pb.CreateWalletResponse, error) {
	return nil, nil
}

func (m *MockWalletClient) GetWallets(_ context.Context, _ *pb.GetWalletsRequest,
	_ ...grpc.CallOption) (*pb.GetWalletsResponse, error) {
	return nil, nil
}

func (m *MockWalletClient) ValidateWalletOwner(ctx context.Context, in *pb.ValidateWalletOwnerRequest,
	_ ...grpc.CallOption) (*pb.ValidateWalletOwnerResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*pb.ValidateWalletOwnerResponse), args.Error(1)
}

func (m *MockWalletClient) FindWalletByAddress(_ context.Context, _ *pb.FindWalletByAddressRequest,
	_ ...grpc.CallOption) (*pb.Wallet, error) {
	return nil, nil
}

func TestDeposit(t *testing.T) {
	mockAssetStore := new(MockAssetStore)
	mockTxStore := new(MockTxStore)
	mockWalletClient := new(MockWalletClient)

	server := Server{
		AssetsStore:  mockAssetStore,
		TxStore:      mockTxStore,
		WalletClient: mockWalletClient,
	}

	walletID := primitive.NewObjectID()
	userID := primitive.NewObjectID().Hex()
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, userID)
	req := &pb.DepositRequest{
		WalletId:  walletID.Hex(),
		AssetName: "BTC",
		Amount:    10.0,
	}

	t.Run("Successful deposit", func(t *testing.T) {
		mockAssetStore.ExpectedCalls = nil
		mockTxStore.ExpectedCalls = nil
		mockWalletClient.ExpectedCalls = nil

		mockWalletClient.On("ValidateWalletOwner", mock.Anything, mock.Anything).
			Return(&pb.ValidateWalletOwnerResponse{IsOwner: true}, nil).Once()
		mockAssetStore.On("UpdateOne", mock.Anything, mock.Anything, mock.Anything).
			Return(&mongo.UpdateResult{}, nil).Once()
		mockTxStore.On("InsertOne", mock.Anything, mock.AnythingOfType("*models.Transaction")).
			Return(&mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil).Once()

		res, err := server.Deposit(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, string(models.StatusCompleted), res.Status)
		mockWalletClient.AssertExpectations(t)
		mockAssetStore.AssertExpectations(t)
		mockTxStore.AssertExpectations(t)
	})

	t.Run("Unauthorized Wallet Interaction", func(t *testing.T) {
		mockAssetStore.ExpectedCalls = nil
		mockTxStore.ExpectedCalls = nil
		mockWalletClient.ExpectedCalls = nil

		mockWalletClient.On("ValidateWalletOwner", mock.Anything, mock.Anything).
			Return(&pb.ValidateWalletOwnerResponse{IsOwner: false}, nil).Once()

		res, err := server.Deposit(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, res)
		mockWalletClient.AssertExpectations(t)
	})
}
