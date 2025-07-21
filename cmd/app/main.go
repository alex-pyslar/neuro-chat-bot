package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv" // Добавлен импорт для godotenv

	"github.com/alex-pyslar/neuro-chat-bot/internal/adapters/llm"
	"github.com/alex-pyslar/neuro-chat-bot/internal/adapters/persistence"
	"github.com/alex-pyslar/neuro-chat-bot/internal/adapters/telegram" // Обновленный путь к Telegram контроллеру
	"github.com/alex-pyslar/neuro-chat-bot/internal/usecases"
	"github.com/alex-pyslar/neuro-chat-bot/pkg/logger"
)

func main() {
	// Загрузка переменных окружения из .env файла
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Инициализация логгера
	appLogger := logger.NewConsoleLogger(logger.AllLevels) // Логируем все уровни

	// Загрузка переменных окружения
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		appLogger.Fatal("MONGO_URI environment variable not set.")
	}
	mongoDBName := os.Getenv("MONGO_DB_NAME")
	if mongoDBName == "" {
		appLogger.Fatal("MONGO_DB_NAME environment variable not set.")
	}
	telegramBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramBotToken == "" {
		appLogger.Fatal("TELEGRAM_BOT_TOKEN environment variable not set.")
	}
	llamaBaseURL := os.Getenv("LLAMA_BASE_URL")
	if llamaBaseURL == "" {
		appLogger.Fatal("LLAMA_BASE_URL environment variable not set. Using default.")
		llamaBaseURL = "http://localhost:8080" // Default for llama-cpp-python server
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Инициализация MongoDB репозитория
	userRepo, err := persistence.NewMongoDbRepository(mongoURI, mongoDBName, appLogger)
	if err != nil {
		appLogger.Fatal("Failed to create MongoDB repository: %v", err)
	}
	appLogger.Info("MongoDB repository initialized.")

	// Инициализация LlamaC++ Gateway
	llamaGateway := llm.NewLlamaCppGateway(llamaBaseURL, appLogger, 60*time.Second)
	appLogger.Info("LlamaC++ Gateway initialized with base URL: %s", llamaBaseURL)

	// Инициализация User Interactor (Use Case)
	userInteractor := usecases.NewUserInteractor(userRepo, llamaGateway, appLogger, 100) // 100 сообщений в истории чата
	appLogger.Info("User Interactor initialized.")

	// Инициализация Telegram Bot Controller
	botController, err := telegram_adapter.NewTelegramBotController(telegramBotToken, appLogger, userInteractor) // Обновленный вызов
	if err != nil {
		appLogger.Fatal("Failed to create Telegram Bot Controller: %v", err)
	}
	appLogger.Info("Telegram Bot Controller initialized.")

	// Запуск polling'а Telegram бота
	appLogger.Info("Starting Telegram Bot Polling...")
	botController.StartPolling(ctx)

	// Ожидание завершения
	<-ctx.Done()
	appLogger.Info("Application shutting down.")
}
