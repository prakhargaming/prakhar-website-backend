package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoDBURI            string
	GeminiAPIKey          string
	SystemPrompt          string
	Port                  string
	MongoVectorDatabase   string
	MongoVectorCollection string
	ClerkWebhookSecret    string
	SendgridAPIKey        string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	promptPath := getEnv("SYSTEM_PROMPT_PATH", "./data/system_prompt.txt")
	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		log.Fatalf("failed to read system prompt at %s: %v", promptPath, err)
	}

	return &Config{
		MongoDBURI:            mustGetEnv("MONGODB_URI"),
		GeminiAPIKey:          mustGetEnv("GEMINI_API_KEY"),
		SystemPrompt:          string(promptBytes),
		Port:                  getEnv("PORT", "8080"),
		MongoVectorDatabase:   mustGetEnv("MONGODB_VECTOR_DATABASE"),
		MongoVectorCollection: mustGetEnv("MONGODB_VECTOR_COLLECTION"),
		ClerkWebhookSecret:    mustGetEnv("CLERK_WEBHOOK_SECRET_PROD"),
		SendgridAPIKey:        mustGetEnv("SENDGRID_API_KEY"),
	}
}

func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return val
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
