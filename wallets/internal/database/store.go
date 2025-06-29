package database

import (
	"context"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/yusufatalay/wallet/wallets/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type WalletStore interface {
	InsertOne(ctx context.Context, wallet *models.Wallet) (*mongo.InsertOneResult, error)
	FindWalletsByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.Wallet, error)
	CountDocuments(ctx context.Context, filter interface{}) (int64, error)
	FindOne(ctx context.Context, filter interface{}) (*models.Wallet, error)
}

type CacheStore interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

type mongoStore struct{ coll *mongo.Collection }

func NewMongoStore(coll *mongo.Collection) WalletStore { return &mongoStore{coll} }

func (s *mongoStore) InsertOne(ctx context.Context, wallet *models.Wallet) (*mongo.InsertOneResult, error) {
	return s.coll.InsertOne(ctx, wallet)
}

func (s *mongoStore) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	return s.coll.CountDocuments(ctx, filter)
}

func (s *mongoStore) FindOne(ctx context.Context, filter interface{}) (*models.Wallet, error) {
	var wallet models.Wallet
	err := s.coll.FindOne(ctx, filter).Decode(&wallet)

	return &wallet, err
}

func (s *mongoStore) FindWalletsByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.Wallet, error) {
	filter := bson.M{"userId": userID}
	cursor, err := s.coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var wallets []models.Wallet
	if err = cursor.All(ctx, &wallets); err != nil {
		return nil, err
	}

	return wallets, nil
}
