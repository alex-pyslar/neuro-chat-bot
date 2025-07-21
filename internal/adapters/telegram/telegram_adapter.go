package telegram_adapter // Имя пакета изменено на telegram_adapter

import (
	"context"
	"fmt"
	"github.com/alex-pyslar/neuro-chat-bot/internal/domain"
	"github.com/alex-pyslar/neuro-chat-bot/pkg/logger"
	telegrambotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv" // Добавлен импорт для strconv
	"strings"
)

// UserInteractorService определяет интерфейс для взаимодействия с UserInteractor.
type UserInteractorService interface {
	GetOrCreateUser(ctx context.Context, userID int64, username string) (*domain.User, error) // Добавлен username
	SaveUser(ctx context.Context, user *domain.User) error
	GetModelResponseForUser(ctx context.Context, user *domain.User, userMessage string) (string, error)
	AddCharacter(ctx context.Context, user *domain.User, newChar *domain.CharacterPreset) error
	ClearChatHistory(ctx context.Context, user *domain.User) error
	UpdateUserProperty(ctx context.Context, user *domain.User, prop string, value string) error
	ChangeCurrentCharacter(ctx context.Context, user *domain.User, index int) error
	ChatHistoryLimit() int
}

// TelegramBotController отвечает за взаимодействие с Telegram API и маршрутизацию запросов.
type TelegramBotController struct {
	botClient   *telegrambotapi.BotAPI
	logger      logger.Logger
	userUseCase UserInteractorService // Зависимость от интерфейса Use Case
}

// NewTelegramBotController создает новый экземпляр TelegramBotController.
func NewTelegramBotController(botToken string, logger logger.Logger, userUseCase UserInteractorService) (*TelegramBotController, error) {
	bot, err := telegrambotapi.NewBotAPI(botToken)
	if err != nil {
		logger.Error("Failed to create new Telegram Bot API: %v", err)
		return nil, fmt.Errorf("failed to create new Telegram Bot API: %w", err)
	}
	bot.Debug = false // Отключите отладочные сообщения в продакшене
	logger.Info("Authorized on account %s", bot.Self.UserName)

	return &TelegramBotController{
		botClient:   bot,
		logger:      logger,
		userUseCase: userUseCase,
	}, nil
}

// StartPolling начинает прослушивание входящих обновлений Telegram.
func (c *TelegramBotController) StartPolling(ctx context.Context) {
	u := telegrambotapi.NewUpdate(0)
	u.Timeout = 60

	updates := c.botClient.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil { // Обработка входящих сообщений
			go c.handleMessage(ctx, update.Message)
		} else if update.CallbackQuery != nil { // Обработка callback-запросов от кнопок
			go c.handleCallbackQuery(ctx, update.CallbackQuery)
		}
	}
}

// handleMessage обрабатывает входящие текстовые сообщения.
func (c *TelegramBotController) handleMessage(ctx context.Context, message *telegrambotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID
	text := message.Text
	username := message.From.UserName
	if username == "" {
		username = fmt.Sprintf("User%d", userID) // Fallback if username is empty
	}

	user, err := c.userUseCase.GetOrCreateUser(ctx, userID, username)
	if err != nil {
		c.logger.Error("Failed to get or create user %d: %v", userID, err)
		c.sendMessage(ctx, chatID, "An error occurred while fetching your data. Please try again later.", nil)
		return
	}

	// Обновляем LastMessageID, если это обычное сообщение
	if user.LastMessageID != 0 {
		c.deleteCommandMessage(ctx, chatID, user.LastMessageID)
		user.LastMessageID = 0 // Сбрасываем после удаления
		if err := c.userUseCase.SaveUser(ctx, user); err != nil {
			c.logger.Error("Failed to save user %d after resetting LastMessageID: %v", userID, err)
		}
	}

	if strings.HasPrefix(text, "/") {
		c.handleCommand(ctx, user, message, chatID, text)
	} else {
		c.handleTextMessage(ctx, user, message, chatID, text)
	}
}

// handleCommand обрабатывает команды бота.
func (c *TelegramBotController) handleCommand(ctx context.Context, user *domain.User, message *telegrambotapi.Message, chatID int64, command string) {
	var response string
	var markup interface{} = nil
	commandHandled := true

	// Сбрасываем pending команду, если пользователь вводит новую команду
	if user.PendingCommand != "" {
		user.PendingCommand = ""
		if err := c.userUseCase.SaveUser(ctx, user); err != nil {
			c.logger.Error("Failed to save user %d after resetting pending command: %v", user.ID, err)
		}
	}

	switch command {
	case "/start":
		response = fmt.Sprintf("Hello, %s! I am your AI assistant. How can I help you today? You can use /menu to see available options.", user.UserName)
	case "/menu":
		response = "What would you like to do?"
		markup = c.createMainMenu()
	case "/newchar":
		newChar := domain.NewCharacterPreset()
		err := c.userUseCase.AddCharacter(ctx, user, newChar)
		if err != nil {
			c.logger.Error("Failed to add new character for user %d: %v", user.ID, err)
			response = "Failed to add new character."
		} else {
			response = fmt.Sprintf("New character '%s' added and set as current.", newChar.Name)
		}
	case "/listchar":
		if len(user.Characters) == 0 {
			response = "You have no characters yet. Use /newchar to create one."
		} else {
			response = "Your characters:\n"
			for i, char := range user.Characters {
				response += fmt.Sprintf("%d. %s %s\n", i+1, char.Name, func() string {
					if i == user.CurrentCharacterID {
						return "(current)"
					}
					return ""
				}())
			}
			response += "\nUse /switchchar <number> to change."
		}
	case "/switchchar":
		user.PendingCommand = "switch_character"
		if err := c.userUseCase.SaveUser(ctx, user); err != nil {
			c.logger.Error("Failed to save user %d after setting pending command: %v", user.ID, err)
		}
		response = "Please enter the number of the character you want to switch to."
	case "/setprompt":
		user.PendingCommand = "set_prompt"
		if err := c.userUseCase.SaveUser(ctx, user); err != nil {
			c.logger.Error("Failed to save user %d after setting pending command: %v", user.ID, err)
		}
		response = "Please enter the new prompt for the current character:"
	case "/setgreeting":
		user.PendingCommand = "set_greeting"
		if err := c.userUseCase.SaveUser(ctx, user); err != nil {
			c.logger.Error("Failed to save user %d after setting pending command: %v", user.ID, err)
		}
		response = "Please enter the new greeting for the current character:"
	case "/setcharname":
		user.PendingCommand = "set_character_name"
		if err := c.userUseCase.SaveUser(ctx, user); err != nil {
			c.logger.Error("Failed to save user %d after setting pending command: %v", user.ID, err)
		}
		response = "Please enter the new name for the current character:"
	case "/setusername":
		user.PendingCommand = "set_user_name"
		if err := c.userUseCase.SaveUser(ctx, user); err != nil {
			c.logger.Error("Failed to save user %d after setting pending command: %v", user.ID, err)
		}
		response = "Please enter your new username:"
	case "/setuserdesc":
		user.PendingCommand = "set_user_description"
		if err := c.userUseCase.SaveUser(ctx, user); err != nil {
			c.logger.Error("Failed to save user %d after setting pending command: %v", user.ID, err)
		}
		response = "Please enter your new description:"
	case "/clearchat":
		err := c.userUseCase.ClearChatHistory(ctx, user)
		if err != nil {
			c.logger.Error("Failed to clear chat history for user %d: %v", user.ID, err)
			response = "Failed to clear chat history."
		} else {
			response = "Chat history cleared."
		}
	case "/charinfo":
		char := user.GetCurrentCharacter()
		response = fmt.Sprintf("<b>Current Character Info:</b>\nName: %s\nGreeting: %s\nPrompt: %s\nChat Messages: %d/%d",
			char.Name, char.Greeting, char.Prompt, len(char.Chat), c.userUseCase.ChatHistoryLimit())
	default:
		response = "Unknown command. Use /menu to see available options."
		commandHandled = false
	}

	if commandHandled {
		c.deleteCommandMessage(ctx, chatID, message.MessageID) // Удаляем сообщение с командой
		sentMessageID := c.sendMessage(ctx, chatID, response, markup)
		if sentMessageID != -1 {
			user.LastMessageID = sentMessageID // Сохраняем ID сообщения бота
			if err := c.userUseCase.SaveUser(ctx, user); err != nil {
				c.logger.Error("Failed to save LastMessageID for user %d: %v", user.ID, err)
			}
		}
	}
}

// handleTextMessage обрабатывает обычные текстовые сообщения (не команды).
func (c *TelegramBotController) handleTextMessage(ctx context.Context, user *domain.User, message *telegrambotapi.Message, chatID int64, text string) {
	var response string
	var err error

	// Если есть ожидающая команда, обрабатываем ее
	if user.PendingCommand != "" {
		response, err = c.handlePendingCommand(ctx, user, text)
		if err != nil {
			c.logger.Error("Error handling pending command for user %d: %v", user.ID, err)
			response = "An error occurred while processing your input. Please try again."
		}
		user.PendingCommand = "" // Сбрасываем ожидающую команду после обработки
		if err := c.userUseCase.SaveUser(ctx, user); err != nil {
			c.logger.Error("Failed to save user %d after handling pending command: %v", user.ID, err)
		}
	} else {
		// Иначе генерируем ответ от модели
		response, err = c.userUseCase.GetModelResponseForUser(ctx, user, text)
		if err != nil {
			c.logger.Error("Error getting model response for user %d: %v", user.ID, err)
			response = "I'm sorry, I couldn't process your request. Please try again."
		}
	}
	sentMessageID := c.sendMessage(ctx, chatID, response, nil)
	if sentMessageID != -1 {
		user.LastMessageID = sentMessageID // Сохраняем ID сообщения бота
		if err := c.userUseCase.SaveUser(ctx, user); err != nil {
			c.logger.Error("Failed to save LastMessageID for user %d: %v", user.ID, err)
		}
	}
}

// handlePendingCommand обрабатывает ввод пользователя в контексте ожидающей команды.
func (c *TelegramBotController) handlePendingCommand(ctx context.Context, user *domain.User, input string) (string, error) {
	switch user.PendingCommand {
	case "switch_character":
		charIndex, err := strconv.Atoi(input)
		if err != nil || charIndex <= 0 || charIndex > len(user.Characters) {
			return "Invalid character number. Please enter a valid number from the list.", nil
		}
		err = c.userUseCase.ChangeCurrentCharacter(ctx, user, charIndex-1) // Индекс начинается с 0
		if err != nil {
			return fmt.Sprintf("Failed to switch character: %v", err), err
		}
		return fmt.Sprintf("Switched to character: %s", user.GetCurrentCharacter().Name), nil
	case "set_prompt":
		err := c.userUseCase.UpdateUserProperty(ctx, user, "Prompt", input)
		if err != nil {
			return fmt.Sprintf("Failed to set prompt: %v", err), err
		}
		return "Prompt updated successfully!", nil
	case "set_greeting":
		err := c.userUseCase.UpdateUserProperty(ctx, user, "Greeting", input)
		if err != nil {
			return fmt.Sprintf("Failed to set greeting: %v", err), err
		}
		return "Greeting updated successfully!", nil
	case "set_character_name":
		err := c.userUseCase.UpdateUserProperty(ctx, user, "CharacterName", input)
		if err != nil {
			return fmt.Sprintf("Failed to set character name: %v", err), err
		}
		return "Character name updated successfully!", nil
	case "set_user_name":
		err := c.userUseCase.UpdateUserProperty(ctx, user, "UserName", input)
		if err != nil {
			return fmt.Sprintf("Failed to set your username: %v", err), err
		}
		return "Your username updated successfully!", nil
	case "set_user_description":
		err := c.userUseCase.UpdateUserProperty(ctx, user, "UserDescription", input)
		if err != nil {
			return fmt.Sprintf("Failed to set your description: %v", err), err
		}
		return "Your description updated successfully!", nil
	default:
		return "Unknown pending command state.", nil
	}
}

// createMainMenu создает клавиатуру с главным меню.
func (c *TelegramBotController) createMainMenu() *telegrambotapi.InlineKeyboardMarkup {
	keyboard := telegrambotapi.NewInlineKeyboardMarkup(
		telegrambotapi.NewInlineKeyboardRow(
			telegrambotapi.NewInlineKeyboardButtonData("New Character", "/newchar"),
			telegrambotapi.NewInlineKeyboardButtonData("List Characters", "/listchar"),
		),
		telegrambotapi.NewInlineKeyboardRow(
			telegrambotapi.NewInlineKeyboardButtonData("Switch Character", "/switchchar"),
			telegrambotapi.NewInlineKeyboardButtonData("Set Character Name", "/setcharname"),
		),
		telegrambotapi.NewInlineKeyboardRow(
			telegrambotapi.NewInlineKeyboardButtonData("Set Prompt", "/setprompt"),
			telegrambotapi.NewInlineKeyboardButtonData("Set Greeting", "/setgreeting"),
		),
		telegrambotapi.NewInlineKeyboardRow(
			telegrambotapi.NewInlineKeyboardButtonData("Set My Name", "/setusername"),
			telegrambotapi.NewInlineKeyboardButtonData("Set My Description", "/setuserdesc"),
		),
		telegrambotapi.NewInlineKeyboardRow(
			telegrambotapi.NewInlineKeyboardButtonData("Clear Chat History", "/clearchat"),
			telegrambotapi.NewInlineKeyboardButtonData("Character Info", "/charinfo"),
		),
	)
	return &keyboard
}

// handleCallbackQuery обрабатывает callback-запросы от инлайн-кнопок.
func (c *TelegramBotController) handleCallbackQuery(ctx context.Context, callbackQuery *telegrambotapi.CallbackQuery) {
	userID := callbackQuery.From.ID
	chatID := callbackQuery.Message.Chat.ID
	command := callbackQuery.Data
	username := callbackQuery.From.UserName
	if username == "" {
		username = fmt.Sprintf("User%d", userID) // Fallback if username is empty
	}

	user, err := c.userUseCase.GetOrCreateUser(ctx, userID, username)
	if err != nil {
		c.logger.Error("Failed to get or create user %d from callback: %v", userID, err)
		c.sendMessage(ctx, chatID, "An error occurred. Please try again.", nil)
		return
	}

	// Обновляем LastMessageID, если это сообщение с меню
	if user.LastMessageID != 0 && user.LastMessageID != callbackQuery.Message.MessageID {
		c.deleteCommandMessage(ctx, chatID, user.LastMessageID)
	}

	// Удаляем сообщение с кнопками, если это не команда "меню" (чтобы оно не висело)
	if command != "/menu" {
		c.deleteCommandMessage(ctx, chatID, callbackQuery.Message.MessageID)
	}

	// Переиспользуем логику обработки команд
	// Важно: в callbackQuery.Message поле Text будет пустым, поэтому используем command
	// также Message.MessageID может быть ID сообщения, на которое ответили, а не ID команды
	// для удаления исходного сообщения с меню, если оно было
	msg := &telegrambotapi.Message{
		MessageID: callbackQuery.Message.MessageID,
		From:      callbackQuery.From,
		Chat:      callbackQuery.Message.Chat,
		Text:      command, // Имитируем текстовую команду
	}
	c.handleCommand(ctx, user, msg, chatID, command)

	// Отвечаем на callback query, чтобы убрать индикатор загрузки на кнопке
	callbackConfig := telegrambotapi.NewCallback(callbackQuery.ID, "")
	_, err = c.botClient.Request(callbackConfig)
	if err != nil {
		c.logger.Error("Failed to answer callback query: %v", err)
	}
}

// sendMessage отправляет сообщение в чат.
func (c *TelegramBotController) sendMessage(ctx context.Context, chatID int64, response string, markup interface{}) int {
	msg := telegrambotapi.NewMessage(chatID, response)
	if markup != nil {
		switch m := markup.(type) {
		case *telegrambotapi.InlineKeyboardMarkup:
			msg.ReplyMarkup = m
		case *telegrambotapi.ReplyKeyboardMarkup:
			msg.ReplyMarkup = m
		}
	}
	msg.ParseMode = telegrambotapi.ModeHTML // Или ModeMarkdown, если вы используете Markdown

	sentMessage, err := c.botClient.Send(msg)
	if err != nil {
		c.logger.Error("Error sending message to chat %d: %v", chatID, err)
		return -1
	}
	return sentMessage.MessageID
}

// deleteCommandMessage удаляет сообщение.
func (c *TelegramBotController) deleteCommandMessage(ctx context.Context, chatID int64, messageID int) {
	deleteConfig := telegrambotapi.NewDeleteMessage(chatID, messageID)
	_, err := c.botClient.Request(deleteConfig)
	if err != nil {
		c.logger.Error("Failed to delete message %d in chat %d: %v", messageID, chatID, err)
	}
}
