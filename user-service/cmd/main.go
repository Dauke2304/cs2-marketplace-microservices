package main

import (
	"context"
	"log"
	"net"

	grpchandler "cs2-marketplace-microservices/user-service/internal/delivery/grpc"
	mongorepo "cs2-marketplace-microservices/user-service/internal/repository/mongo"
	"cs2-marketplace-microservices/user-service/internal/usecase"
	config "cs2-marketplace-microservices/user-service/pkg"
	"cs2-marketplace-microservices/user-service/pkg/email"
	"cs2-marketplace-microservices/user-service/pkg/messaging"
	userpb "cs2-marketplace-microservices/user-service/proto/user"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

func main() {

	cfg := config.Load()

	// Initialize MongoDB
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	// Verify MongoDB connection
	err = mongoClient.Ping(context.Background(), nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	// Initialize NATS
	natsClient, err := messaging.New("nats://localhost:4222")
	if err != nil {
		log.Fatalf("NATS connection failed: %v", err)
	}
	defer func() {
		log.Println("Closing NATS connection")
		natsClient.Conn.Close()
	}()

	// Initialize repositories
	db := mongoClient.Database(cfg.MongoDBName)
	userRepo := mongorepo.NewUserRepository(db)
	sessionRepo := mongorepo.NewSessionRepository(db)
	tokenRepo := mongorepo.NewPasswordResetTokenRepository(db)

	// Initialize use cases
	emailSender := email.NewGMailSender(cfg.EmailUser, cfg.EmailPassword)
	// Initialize use cases
	userUC := usecase.NewUserUseCase(
		userRepo,
		sessionRepo,
		tokenRepo,
		emailSender,
		natsClient,
	)

	// Create gRPC server

	grpcServer := grpc.NewServer()
	userHandler := grpchandler.NewUserHandler(*userUC)
	userpb.RegisterUserServiceServer(grpcServer, userHandler)

	// Start server
	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
