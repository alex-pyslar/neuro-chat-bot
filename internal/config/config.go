package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config содержит все настройки приложения
type Config struct {
	Telegram TelegramConfig
	MongoDB  MongoDBConfig
	LlamaCPP LlamaCPPConfig
	Chat     ChatConfig
}

// TelegramConfig настройки для Telegram бота
type TelegramConfig struct {
	BotToken string
	Debug    bool
}

// MongoDBConfig настройки для MongoDB
type MongoDBConfig struct {
	ConnectionString string
	DatabaseName     string
}

// LlamaCPPConfig настройки для Llama.cpp gateway
type LlamaCPPConfig struct {
	BaseURL        string
	TimeoutSeconds int
}

// ChatConfig настройки для логики чата
type ChatConfig struct {
	HistoryLimit int
}

// LoadConfig загружает конфигурацию из переменных окружения.
func LoadConfig() (*Config, error) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return nil, fmt.Errorf("MONGO_URI environment variable not set")
	}
	mongoDBName := os.Getenv("MONGO_DB_NAME")
	if mongoDBName == "" {
		return nil, fmt.Errorf("MONGO_DB_NAME environment variable not set")
	}

	llamaBaseURL := os.Getenv("LLAMA_BASE_URL")
	if llamaBaseURL == "" {
		return nil, fmt.Errorf("LLAMA_BASE_URL environment variable not set")
	}

	llamaTimeoutStr := os.Getenv("LLAMA_TIMEOUT_SECONDS")
	llamaTimeout, err := strconv.Atoi(llamaTimeoutStr)
	if err != nil {
		llamaTimeout = 30 // Дефолтное значение
	}

	chatHistoryLimitStr := os.Getenv("CHAT_HISTORY_LIMIT")
	chatHistoryLimit, err := strconv.Atoi(chatHistoryLimitStr)
	if err != nil {
		chatHistoryLimit = 10 // Дефолтное значение
	}

	debugStr := os.Getenv("TELEGRAM_DEBUG")
	debug := false
	if debugStr == "true" {
		debug = true
	}

	return &Config{
		Telegram: TelegramConfig{
			BotToken: botToken,
			Debug:    debug,
		},
		MongoDB: MongoDBConfig{
			ConnectionString: mongoURI,
			DatabaseName:     mongoDBName,
		},
		LlamaCPP: LlamaCPPConfig{
			BaseURL:        llamaBaseURL,
			TimeoutSeconds: llamaTimeout,
		},
		Chat: ChatConfig{
			HistoryLimit: chatHistoryLimit,
		},
	}, nil
}
