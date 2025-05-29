package config

import (
	"os"
)

// getEnv retrieves an environment variable or returns a fallback default.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

type Config struct {
	MongoURI      string
	MongoDBName   string
	GRPCPort      string
	JWTSecret     string
	EmailUser     string
	EmailPassword string
}

func Load() *Config {
	return &Config{
		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName: getEnv("MONGO_DB", "user_service"),
		GRPCPort:    getEnv("GRPC_PORT", ":50052"),
		JWTSecret:   getEnv("JWT_SECRET", "secret"),
	}
}
