package main

import (
	"context" // Added missing import
	"log"
	"net"

	deliveryGrpc "cs2-marketplace-microservices/inventory-service/internal/delivery/grpc"
	"cs2-marketplace-microservices/inventory-service/internal/repository/mongo"
	"cs2-marketplace-microservices/inventory-service/internal/usecase"
	"cs2-marketplace-microservices/inventory-service/pkg/database"
	"cs2-marketplace-microservices/inventory-service/proto/inventory" // Added proto import

	grpcServer "google.golang.org/grpc"
)

func main() {
	// 1. Init DB
	client, err := database.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	// 2. Setup layers
	repo := mongo.NewInventoryRepository(client.Database("cs2_skins"))
	uc := usecase.NewInventoryUsecase(repo)
	handler := deliveryGrpc.NewHandler(*uc)

	// 3. Start gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}
	s := grpcServer.NewServer()
	inventory.RegisterInventoryServiceServer(s, handler)
	log.Println("Inventory Service running on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
