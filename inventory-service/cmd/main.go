package main

import (
	"context"
	"log"
	"net"
	"net/http"

	deliveryGrpc "cs2-marketplace-microservices/inventory-service/internal/delivery/grpc"
	"cs2-marketplace-microservices/inventory-service/internal/repository/mongo"
	"cs2-marketplace-microservices/inventory-service/internal/usecase"
	"cs2-marketplace-microservices/inventory-service/pkg/database"
	"cs2-marketplace-microservices/inventory-service/pkg/messaging"
	"cs2-marketplace-microservices/inventory-service/pkg/metrics"
	"cs2-marketplace-microservices/inventory-service/proto/inventory"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	grpcServer "google.golang.org/grpc"
)

func main() {
	// Start metrics server in a separate goroutine
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", healthCheckHandler)
		log.Println("Metrics server running on :8082")
		if err := http.ListenAndServe(":8082", nil); err != nil {
			log.Printf("Metrics server failed: %v", err)
		}
	}()

	// 1. Init DB
	client, err := database.InitDB()
	if err != nil {
		log.Fatal(err)
		metrics.ServiceUp.Set(0)
	}
	defer client.Disconnect(context.Background())

	// Set database connection metric
	metrics.DatabaseConnections.Set(1)

	// 4. Init NATS
	natsClient, err := messaging.New("nats://localhost:4222")
	if err != nil {
		log.Fatalf("NATS connection failed: %v", err)
		metrics.ServiceUp.Set(0)
	}
	defer natsClient.Conn.Close()

	// 2. Setup layers
	repo := mongo.NewInventoryRepository(client.Database("cs2_skins"))
	uc := usecase.NewInventoryUsecase(repo, natsClient)
	handler := deliveryGrpc.NewHandler(*uc)

	// 3. Start gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
		metrics.ServiceUp.Set(0)
	}
	s := grpcServer.NewServer()
	inventory.RegisterInventoryServiceServer(s, handler)

	log.Println("Inventory Service running on :50051")
	log.Println("Metrics available at http://localhost:8082/metrics")
	log.Println("Health check available at http://localhost:8082/health")

	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
		metrics.ServiceUp.Set(0)
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
