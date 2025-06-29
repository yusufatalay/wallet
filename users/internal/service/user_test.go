package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	pb "github.com/yusufatalay/wallet/proto"
	"github.com/yusufatalay/wallet/users/internal/models"
	"go.mongodb.org/mongo-driver/mongo"
)

// MockUserStore implements the database.UserStore interface for testing.
type MockUserStore struct {
	mock.Mock
}

func (m *MockUserStore) InsertOne(ctx context.Context, user *models.User) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*mongo.InsertOneResult), args.Error(1)
}

func (m *MockUserStore) FindOneByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*models.User), args.Error(1)
}

func TestRegister(t *testing.T) {
	mockStore := new(MockUserStore)
	server := Server{
		Store:     mockStore,
		JwtSecret: "test-secret",
	}

	t.Run("Successful registration", func(t *testing.T) {
		mockStore.On("InsertOne", mock.Anything, mock.AnythingOfType("*models.User")).
			Return(&mongo.InsertOneResult{}, nil).Once()
		req := &pb.RegisterRequest{Email: "test@test.com", Password: "password"}

		res, err := server.Register(context.Background(), req)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		mockStore.AssertExpectations(t)
	})

	t.Run("Database error", func(t *testing.T) {
		mockStore.On("InsertOne", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil,
			errors.New("db error")).Once()
		req := &pb.RegisterRequest{Email: "error@test.com", Password: "password"}

		res, err := server.Register(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})
}
