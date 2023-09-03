package cli

import (
	"context"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var clientMongo *mongo.Client

// Go routines safe mongo.Client Singleton
func GetMongoClient() *mongo.Client {
	if clientMongo == nil {
		lock.Lock()
		defer lock.Unlock()

		if clientMongo == nil {
			ctx := context.Background()

			client, err := mongo.Connect(
				ctx,
				options.Client().ApplyURI(os.Getenv("MONGO_URI")),
			)
			if err != nil {
				if err = client.Ping(ctx, readpref.Primary()); err != nil {
					log.Fatal(err)
				}
			}

			return client
		} else {
			return clientMongo
		}

	} else {
		return clientMongo
	}
}

// Get collection from name
func GetCollection(name string) *mongo.Collection {
	db := GetMongoClient().Database(os.Getenv("MONGO_DATABASE"))
	return db.Collection(name)
}

// Transaction Handling funcitons
func StartTransaction(ctx context.Context) (mongo.Session, error) {
	session, err := GetMongoClient().StartSession()
	if err != nil {
		return nil, err
	}

	err = session.StartTransaction()
	if err != nil {
		session.EndSession(ctx)
		return nil, err
	}

	return session, nil
}

func CommitTransaction(ctx context.Context, session mongo.Session) error {
	err := session.CommitTransaction(ctx)
	if err != nil {
		session.AbortTransaction(ctx)
		session.EndSession(ctx)
		return err
	}

	session.EndSession(ctx)
	return nil
}
