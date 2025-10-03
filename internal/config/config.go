package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	WhatsAppAPIURL   string
	WhatsAppUsername string
	WhatsAppPassword string
	WhatsAppPath     string
	WhatsappWebhookSecret string
	OpenAIAPIKey     string
	ServerPort       string
	SessionTimeout   int
	CacheTTL         int
}

func Load() *Config {
	// Load .env file if exists
	godotenv.Load()

	return &Config{
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/task_manager"),
		RedisURL:         getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:        getEnv("JWT_SECRET", "your_jwt_secret"),
		WhatsAppAPIURL:   getEnv("WHATSAPP_API_URL", "https://whatsapp-go.sebagja.id"),
		WhatsAppUsername: getEnv("WHATSAPP_USERNAME", "your_whatsapp_username"),
		WhatsAppPassword: getEnv("WHATSAPP_PASSWORD", "your_whatsapp_password"),
		WhatsAppPath:     getEnv("WHATSAPP_PATH", "your_whatsapp_path"),
		WhatsappWebhookSecret: getEnv("WHATSAPP_WEBHOOK_SECRET", "superadmin"),
		OpenAIAPIKey:     getEnv("OPENAI_API_KEY", "your_openai_api_key"),
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		SessionTimeout:   getEnvAsInt("SESSION_TIMEOUT", 3600),
		CacheTTL:         getEnvAsInt("CACHE_TTL", 1800),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
