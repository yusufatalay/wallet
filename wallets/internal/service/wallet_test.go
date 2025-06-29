package service

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yusufatalay/wallet/pkg/middleware"
	pb "github.com/yusufatalay/wallet/proto"
	"github.com/yusufatalay/wallet/wallets/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MockStore struct{ mock.Mock }

func (m *MockStore) InsertOne(ctx context.Context, wallet *models.Wallet) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, wallet)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*mongo.InsertOneResult), args.Error(1)
}

func (m *MockStore) FindWalletsByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.Wallet, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]models.Wallet), args.Error(1)
}

func (m *MockStore) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	args := m.Called(ctx, filter)

	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStore) FindOne(ctx context.Context, filter interface{}) (*models.Wallet, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*models.Wallet), args.Error(1)
}

func (m *MockStore) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)

	return args.Get(0).(*redis.StringCmd)
}

func (m *MockStore) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)

	return args.Get(0).(*redis.StatusCmd)
}

func (m *MockStore) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	args := m.Called(ctx, keys)

	return args.Get(0).(*redis.IntCmd)
}

func TestGetWallets(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, primitive.NewObjectID().Hex())

	t.Run("Cache Miss Scenario", func(t *testing.T) {
		mockStore := new(MockStore)
		server := Server{Store: mockStore, Cache: mockStore}

		// Setup
		mockStore.On("Get", mock.Anything, mock.AnythingOfType("string")).
			Return(redis.NewStringResult("", redis.Nil)).Once()

		mockStore.On("FindWalletsByUserID", mock.Anything, mock.AnythingOfType("primitive.ObjectID")).
			Return([]models.Wallet{}, nil).Once()

		mockStore.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything,
			mock.AnythingOfType("time.Duration")).Return(redis.NewStatusResult("", nil)).Once()

		_, err := server.GetWallets(ctx, &pb.GetWalletsRequest{})

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("Cache Hit Scenario", func(t *testing.T) {
		mockStore := new(MockStore)
		server := Server{Store: mockStore, Cache: mockStore}

		cachedJSON := `[{"walletId":"fakeid","userId":"fakeuserid"}]`
		mockStore.On("Get", mock.Anything, mock.AnythingOfType("string")).
			Return(redis.NewStringResult(cachedJSON, nil)).Once()

		res, err := server.GetWallets(ctx, &pb.GetWalletsRequest{})

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.Wallets, 1)
		mockStore.AssertExpectations(t)
	})
}

func TestCreateWallet(t *testing.T) {
	mockStore := new(MockStore)
	server := Server{Store: mockStore, Cache: mockStore}
	ctx := context.WithValue(context.Background(), middleware.UserIDKey, primitive.NewObjectID().Hex())

	t.Run("Successful wallet creation", func(t *testing.T) {
		req := &pb.CreateWalletRequest{Address: "test-addr", Network: "BTC"}
		mockStore.On("InsertOne", mock.Anything, mock.AnythingOfType("*models.Wallet")).
			Return(&mongo.InsertOneResult{InsertedID: primitive.NewObjectID()}, nil).Once()
		mockStore.On("Del", mock.Anything, mock.Anything).Return(redis.NewIntResult(1, nil)).Once()

		res, err := server.CreateWallet(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "test-addr", res.Wallet.Address)
		mockStore.AssertExpectations(t)
	})
}
