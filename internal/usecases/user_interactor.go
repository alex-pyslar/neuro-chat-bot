package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/alex-pyslar/neuro-chat-bot/internal/domain"
	"github.com/alex-pyslar/neuro-chat-bot/pkg/logger"
)

// UserRepository определяет интерфейс для сохранения и загрузки пользователей.
// Этот интерфейс находится в слое Use Cases, но его реализация будет в Adapters/Persistence.
type UserRepository interface {
	SaveUser(ctx context.Context, user *domain.User) error
	LoadUser(ctx context.Context, userID int64) (*domain.User, error)
	AddChatMessage(ctx context.Context, userID int64, characterIndex int, message domain.ChatMessage) error
}

// ModelGateway определяет интерфейс для взаимодействия с моделью ИИ.
// Этот интерфейс находится в слое Use Cases, но его реализация будет в Adapters/LLM.
type ModelGateway interface {
	GetModelResponse(ctx context.Context, messages []domain.ChatMessage, config ModelConfig) (string, error)
}

// ModelConfig содержит параметры для запроса к модели.
type ModelConfig struct {
	MaxTokens        int
	Temperature      float64
	MinP             float64
	TopP             float64
	TopK             float64
	RepeatPenalty    float64
	PresencePenalty  float64
	FrequencyPenalty float64
	// StopSequences []string
}

// UserInteractor содержит бизнес-логику, связанную с пользователями и чатом.
type UserInteractor struct {
	userRepo         UserRepository
	modelGateway     ModelGateway
	logger           logger.Logger
	chatHistoryLimit int
}

// NewUserInteractor создает новый экземпляр UserInteractor.
func NewUserInteractor(userRepo UserRepository, modelGateway ModelGateway, logger logger.Logger, chatHistoryLimit int) *UserInteractor {
	return &UserInteractor{
		userRepo:         userRepo,
		modelGateway:     modelGateway,
		logger:           logger,
		chatHistoryLimit: chatHistoryLimit,
	}
}

// GetOrCreateUser загружает существующего пользователя или создает нового.
func (uc *UserInteractor) GetOrCreateUser(ctx context.Context, userID int64, username string) (*domain.User, error) {
	user, err := uc.userRepo.LoadUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	if user == nil {
		user = domain.NewUser(userID, username)
		if err := uc.userRepo.SaveUser(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to save new user: %w", err)
		}
		uc.logger.Info("Created new user with ID: %d", userID)
	} else {
		// Update username if it changed
		if user.UserName != username {
			user.UserName = username
			if err := uc.userRepo.SaveUser(ctx, user); err != nil {
				uc.logger.Error("Failed to update username for user %d: %v", userID, err)
			}
		}
	}
	user.RequestTime = time.Now() // Обновляем время запроса
	return user, nil
}

// SaveUser сохраняет данные пользователя.
func (uc *UserInteractor) SaveUser(ctx context.Context, user *domain.User) error {
	return uc.userRepo.SaveUser(ctx, user)
}

// GetModelResponseForUser генерирует ответ модели для пользователя.
func (uc *UserInteractor) GetModelResponseForUser(ctx context.Context, user *domain.User, userMessage string) (string, error) {
	currentChatIndex := user.CurrentCharacterID
	currentChat := user.GetCurrentCharacter().Chat

	// Добавляем сообщение пользователя в историю
	user.GetCurrentCharacter().Chat = append(currentChat, domain.NewChatMessage(domain.UserRole, userMessage))
	user.EnsureChatHistoryLimit(currentChatIndex, uc.chatHistoryLimit) // Обрезаем историю
	if err := uc.userRepo.SaveUser(ctx, user); err != nil {
		uc.logger.Error("Failed to save user after adding message: %v", err)
		return "", fmt.Errorf("failed to save chat message: %w", err)
	}

	// Подготовка сообщений для модели
	messagesForModel := user.GetCurrentCharacter().GetChatMessagesForModel()
	messagesForModel = uc.applyPlaceholdersToMessages(messagesForModel, user) // Применяем плейсхолдеры

	// Параметры для модели (можно сделать настраиваемыми)
	modelConfig := ModelConfig{
		MaxTokens:        500,
		Temperature:      0.7,
		TopP:             0.9,
		TopK:             0, // 0 отключает TopK
		RepeatPenalty:    1.1,
		PresencePenalty:  0.0,
		FrequencyPenalty: 0.0,
	}

	response, err := uc.modelGateway.GetModelResponse(ctx, messagesForModel, modelConfig)
	if err != nil {
		uc.logger.Error("Failed to get model response: %v", err)
		return "", fmt.Errorf("failed to get model response: %w", err)
	}

	// Добавляем ответ модели в историю
	user.GetCurrentCharacter().Chat = append(user.GetCurrentCharacter().Chat, domain.NewChatMessage(domain.Assistant, response))
	user.EnsureChatHistoryLimit(currentChatIndex, uc.chatHistoryLimit) // Обрезаем историю после добавления ответа
	if err := uc.userRepo.SaveUser(ctx, user); err != nil {
		uc.logger.Error("Failed to save user after adding model response: %v", err)
		return "", fmt.Errorf("failed to save model response: %w", err)
	}

	return response, nil
}

// AddCharacter добавляет нового персонажа для пользователя и делает его текущим.
func (uc *UserInteractor) AddCharacter(ctx context.Context, user *domain.User, newChar *domain.CharacterPreset) error {
	// Присваиваем ID новому персонажу (простой инкремент)
	newChar.ID = len(user.Characters) // Простое присвоение ID на основе количества существующих персонажей
	user.Characters = append(user.Characters, newChar)
	user.ChangeCurrentCharacter(len(user.Characters) - 1)
	return uc.userRepo.SaveUser(ctx, user)
}

// ClearChatHistory clears the chat history for the current character.
func (uc *UserInteractor) ClearChatHistory(ctx context.Context, user *domain.User) error {
	user.GetCurrentCharacter().Chat = make([]domain.ChatMessage, 0, uc.chatHistoryLimit)
	return uc.userRepo.SaveUser(ctx, user)
}

// UpdateUserProperty updates a string property of the user and saves it.
func (uc *UserInteractor) UpdateUserProperty(ctx context.Context, user *domain.User, prop string, value string) error {
	switch prop {
	case "Prompt":
		user.GetCurrentCharacter().Prompt = user.ReplacePlaceholders(value)
	case "UserName":
		user.UserName = value
	case "UserDescription":
		user.UserDescription = user.ReplacePlaceholders(value)
	case "CharacterName":
		user.GetCurrentCharacter().Name = value
	case "Greeting":
		user.GetCurrentCharacter().Greeting = user.ReplacePlaceholders(value)
	default:
		return fmt.Errorf("unknown user property: %s", prop)
	}
	return uc.userRepo.SaveUser(ctx, user)
}

// ChangeCurrentCharacter changes the active character for the user.
func (uc *UserInteractor) ChangeCurrentCharacter(ctx context.Context, user *domain.User, index int) error {
	if index < 0 || index >= len(user.Characters) {
		return fmt.Errorf("invalid character index: %d", index)
	}
	user.ChangeCurrentCharacter(index)
	return uc.userRepo.SaveUser(ctx, user)
}

// ChatHistoryLimit возвращает текущий лимит истории чата.
func (uc *UserInteractor) ChatHistoryLimit() int {
	return uc.chatHistoryLimit
}

// applyPlaceholdersToMessages применяет плейсхолдеры к сообщениям.
func (uc *UserInteractor) applyPlaceholdersToMessages(messages []domain.ChatMessage, user *domain.User) []domain.ChatMessage {
	processedMessages := make([]domain.ChatMessage, len(messages))
	for i, msg := range messages {
		processedContent := user.ReplacePlaceholders(msg.Content)
		// Проверяем, нужно ли применять плейсхолдеры для персонажа, если это не системное сообщение
		if msg.ERole != domain.System {
			processedContent = user.GetCurrentCharacter().ReplacePlaceholders(processedContent)
		}
		processedMessages[i] = domain.NewChatMessage(msg.ERole, processedContent)
	}
	return processedMessages
}
