package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Get MongoDB URI from environment variable
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI environment variable is required")
	}

	database := os.Getenv("MONGO_DATABASE")
	if database == "" {
		database = "ethereum_raw_data"
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	db := client.Database(database)

	fmt.Printf("🗑️  Clearing MongoDB database: %s\n", database)

	// Clear collections
	collections := []string{"blocks", "transactions", "crawler_metrics", "system_health"}

	for _, collectionName := range collections {
		collection := db.Collection(collectionName)

		// Drop the collection
		if err := collection.Drop(ctx); err != nil {
			log.Printf("Warning: Failed to drop collection %s: %v", collectionName, err)
		} else {
			fmt.Printf("✅ Cleared collection: %s\n", collectionName)
		}
	}

	fmt.Println("🎯 Database reset complete! You can now start crawler from configured START_BLOCK_NUMBER")
}
