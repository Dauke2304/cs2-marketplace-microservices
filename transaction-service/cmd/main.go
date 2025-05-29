package main

import (
	grpcDelivery "cs2-marketplace-microservices/transaction-service/internal/delivery/grpc"
	"cs2-marketplace-microservices/transaction-service/internal/repository"
	repomongo "cs2-marketplace-microservices/transaction-service/internal/repository/mongo"
	"cs2-marketplace-microservices/transaction-service/internal/usecase"
	"cs2-marketplace-microservices/transaction-service/pkg/config"
	"cs2-marketplace-microservices/transaction-service/pkg/database"
	"cs2-marketplace-microservices/transaction-service/proto/transaction"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	db, err := database.InitDB(cfg.MongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.CloseDB()

	// Initialize repository
	transactionRepo := repomongo.NewTransactionRepository(db)
	repositories := repository.NewRepositories(transactionRepo)

	// Initialize use case
	transactionUsecase := usecase.NewTransactionUsecase(repositories.Transaction)

	// Initialize gRPC handler
	handler := grpcDelivery.NewHandler(transactionUsecase)

	// Create gRPC server
	server := grpc.NewServer()

	// Register the transaction service
	transaction.RegisterTransactionServiceServer(server, handler)

	// Enable reflection for testing with tools like grpcurl
	reflection.Register(server)

	// Listen on the configured port
	listener, err := net.Listen("tcp", cfg.ServerPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.ServerPort, err)
	}

	log.Printf("Transaction service starting on port %s", cfg.ServerPort)
	log.Printf("Connected to MongoDB: %s", cfg.MongoURI)
	log.Printf("Database: %s", cfg.DBName)

	// Start the gRPC server
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}
