package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI   string
	ServerPort string
	DBName     string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		MongoURI:   getEnv("MONGO_URI", "mongodb://localhost:27017"),
		ServerPort: getEnv("SERVER_PORT", ":50053"),
		DBName:     getEnv("DB_NAME", "cs2_transactions"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
