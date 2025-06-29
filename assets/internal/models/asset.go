package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Asset represents an asset in db.
type Asset struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	WalletID primitive.ObjectID `bson:"walletId"`
	Name     string             `bson:"name"`
	Amount   float64            `bson:"amount"`
}
