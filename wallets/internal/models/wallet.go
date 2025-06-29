package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Wallet, represents wallet document in db.
type Wallet struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	UserID  primitive.ObjectID `bson:"userId"`
	Address string             `bson:"address"`
	Network string             `bson:"network"`
}
