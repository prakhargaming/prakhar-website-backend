package database

import (
	"log"
	"sync"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	once   sync.Once
	client *mongo.Client
)

func Init(uri string) *mongo.Client {
	once.Do(func() {
		var err error
		client, err = mongo.Connect(options.Client().ApplyURI(uri))
		if err != nil {
			log.Fatalf("failed to connect to MongoDB: %v", err)
		}
	})
	return client
}
