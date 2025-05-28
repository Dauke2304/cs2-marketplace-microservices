package main

import (
	"context"
	"cs2-marketplace-microservices/inventory-service/pkg/database"
	"log"
)

func main() {
	client, err := database.InitDB()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer client.Disconnect(context.Background())
	log.Println("Successfully connected to MongoDB!")
}
