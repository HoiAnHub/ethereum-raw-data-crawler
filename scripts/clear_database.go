package main

import (
	"context"
	"ethereum-raw-data-crawler/internal/infrastructure/config"
	"ethereum-raw-data-crawler/internal/infrastructure/database"
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Println("Clearing database...")

	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Connect to MongoDB
	db, err := database.NewMongoDB(&cfg.MongoDB)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Clear collections (focus on blocks and transactions due to schema change)
	collections := []string{"blocks", "transactions"}

	for _, collName := range collections {
		coll := db.GetCollection(collName)
		result, err := coll.DeleteMany(ctx, map[string]interface{}{})
		if err != nil {
			log.Printf("Failed to clear collection %s: %v", collName, err)
		} else {
			fmt.Printf("Cleared %d documents from collection: %s\n", result.DeletedCount, collName)
		}
	}

	fmt.Println("Database cleared successfully!")
}
