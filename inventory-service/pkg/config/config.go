package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI string
	NATSURL  string `envconfig:"NATS_URL" default:"nats://localhost:4222"`
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		MongoURI: getEnv("MONGO_URI", "mongodb://localhost:27017/cs2_skins_marketplace"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
