package database

import (
	"context"

	"github.com/yusufatalay/wallet/users/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserStore defines the methods our service needs for database interactions.
type UserStore interface {
	InsertOne(ctx context.Context, user *models.User) (*mongo.InsertOneResult, error)
	FindOneByEmail(ctx context.Context, email string) (*models.User, error)
}

// mongoStore implements the UserStore interface.
type mongoStore struct {
	coll *mongo.Collection
}

// NewUserStore creates a new mongoStore.
func NewUserStore(coll *mongo.Collection) UserStore {
	return &mongoStore{coll: coll}
}

func (s *mongoStore) InsertOne(ctx context.Context, user *models.User) (*mongo.InsertOneResult, error) {
	return s.coll.InsertOne(ctx, user)
}

func (s *mongoStore) FindOneByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := s.coll.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
