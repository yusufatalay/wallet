package database

import (
	"context"

	"github.com/yusufatalay/wallet/assets/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AssetStore defines the methods for asset database operations.
type AssetStore interface {
	FindOne(ctx context.Context, filter interface{}) (*models.Asset, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{},
		opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	FindAssetsByWalletID(ctx context.Context, walletID primitive.ObjectID) ([]models.Asset, error)
}

// TransactionStore defines the methods for transaction database operations.
type TransactionStore interface {
	InsertOne(ctx context.Context, tx *models.Transaction) (*mongo.InsertOneResult, error)
	Find(ctx context.Context, filter interface{}) ([]models.Transaction, error)
}

// mongoAssetStore implements the AssetStore interface.
type mongoAssetStore struct{ coll *mongo.Collection }

func NewAssetStore(coll *mongo.Collection) AssetStore { return &mongoAssetStore{coll} }

func (s *mongoAssetStore) FindOne(ctx context.Context, filter interface{}) (*models.Asset, error) {
	var asset models.Asset
	err := s.coll.FindOne(ctx, filter).Decode(&asset)

	return &asset, err
}

func (s *mongoAssetStore) UpdateOne(ctx context.Context, filter interface{}, update interface{},
	opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return s.coll.UpdateOne(ctx, filter, update, opts...)
}

func (s *mongoAssetStore) FindAssetsByWalletID(ctx context.Context, walletID primitive.ObjectID) ([]models.Asset,
	error) {
	filter := bson.M{"walletId": walletID}
	cursor, err := s.coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var assets []models.Asset
	if err = cursor.All(ctx, &assets); err != nil {
		return nil, err
	}

	return assets, nil
}

// mongoTransactionStore implements the TransactionStore interface.
type mongoTransactionStore struct{ coll *mongo.Collection }

func NewTransactionStore(coll *mongo.Collection) TransactionStore {
	return &mongoTransactionStore{coll}
}

func (s *mongoTransactionStore) InsertOne(ctx context.Context, tx *models.Transaction) (*mongo.InsertOneResult, error) {
	return s.coll.InsertOne(ctx, tx)
}

func (s *mongoTransactionStore) Find(ctx context.Context, filter interface{}) ([]models.Transaction, error) {
	cursor, err := s.coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var transactions []models.Transaction
	if err = cursor.All(ctx, &transactions); err != nil {
		return nil, err
	}

	return transactions, nil
}
