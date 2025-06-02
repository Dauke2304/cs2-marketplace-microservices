package main

import (
	"context"
	"log"
	"net"
	"net/http"

	grpchandler "cs2-marketplace-microservices/user-service/internal/delivery/grpc"
	mongorepo "cs2-marketplace-microservices/user-service/internal/repository/mongo"
	"cs2-marketplace-microservices/user-service/internal/usecase"
	config "cs2-marketplace-microservices/user-service/pkg"
	"cs2-marketplace-microservices/user-service/pkg/email"
	"cs2-marketplace-microservices/user-service/pkg/messaging"
	"cs2-marketplace-microservices/user-service/pkg/metrics"
	userpb "cs2-marketplace-microservices/user-service/proto/user"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

func main() {
	// Start metrics server in a separate goroutine
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", healthCheckHandler)
		log.Println("User Service metrics server running on :8081")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Printf("Metrics server failed: %v", err)
		}
	}()

	cfg := config.Load()

	// Initialize MongoDB
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
		metrics.ServiceUp.Set(0)
	}
	defer mongoClient.Disconnect(context.Background())

	// Verify MongoDB connection
	err = mongoClient.Ping(context.Background(), nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
		metrics.ServiceUp.Set(0)
	}

	// Set database connection metric
	metrics.DatabaseConnections.Set(1)

	// Initialize NATS
	natsClient, err := messaging.New("nats://localhost:4222")
	if err != nil {
		log.Fatalf("NATS connection failed: %v", err)
		metrics.ServiceUp.Set(0)
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
		metrics.ServiceUp.Set(0)
	}

	log.Printf("User Service running at %v", lis.Addr())
	log.Println("Metrics available at http://localhost:8081/metrics")
	log.Println("Health check available at http://localhost:8081/health")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
		metrics.ServiceUp.Set(0)
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
