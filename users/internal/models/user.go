package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// User,  represents user document in db.
type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Email    string             `bson:"email"`
	Password string             `bson:"password"`
}
