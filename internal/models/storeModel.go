package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Store struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id"`
	Name      string             `json:"name" bson:"name"`
	Type      string             `json:"type" bson:"type"`
	DateAdded time.Time          `json:"date_added" bson:"date_added"`
}

type QueueMessage struct {
	Index  int
	Buffer []byte
}
