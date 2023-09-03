package repositories

import (
	"context"

	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/cli"
	"github.com/Simone-Convertini/Speech-Summarization-Demo/internal/models"
	"go.mongodb.org/mongo-driver/mongo"
)

type StoreRepository struct {
	Context context.Context
}

// Handling Collection:
var collection *mongo.Collection

func getCollection() *mongo.Collection {
	if collection == nil {
		collection = cli.GetCollection("store")
		return collection
	}
	return collection
}

// Repo Operations
func (sr *StoreRepository) InsertStore(store models.Store) (*mongo.InsertOneResult, error) {
	res, err := getCollection().InsertOne(sr.Context, store)
	return res, err
}
