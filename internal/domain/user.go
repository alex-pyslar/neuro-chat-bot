package domain

import (
	"strings"
	"time"
)

// User представляет пользователя бота.
type User struct {
	ID                 int64              `json:"id" bson:"_id"` // Идентификатор пользователя в Telegram
	UserName           string             `json:"user_name" bson:"user_name"`
	UserDescription    string             `json:"user_description" bson:"user_description"`
	Characters         []*CharacterPreset `json:"characters" bson:"characters"` // Список настроек персонажей пользователя
	CurrentCharacterID int                `json:"current_character_id" bson:"current_character_id"`
	RequestTime        time.Time          `json:"request_time" bson:"request_time"`       // Время последнего запроса (для контроля частоты)
	PendingCommand     string             `json:"pending_command" bson:"pending_command"` // Ожидаемая команда (например, для ввода Prompt)
	LastMessageID      int                `json:"last_message_id" bson:"last_message_id"` // ID последнего сообщения бота пользователю
}

// NewUser создает новый экземпляр User с настройками по умолчанию.
func NewUser(userID int64, username string) *User {
	defaultChar := NewCharacterPreset()
	return &User{
		ID:                 userID,
		UserName:           username,
		UserDescription:    "",
		Characters:         []*CharacterPreset{defaultChar},
		CurrentCharacterID: 0,
		RequestTime:        time.Now(),
		PendingCommand:     "",
		LastMessageID:      0,
	}
}

// GetCurrentCharacter возвращает текущего персонажа пользователя.
func (u *User) GetCurrentCharacter() *CharacterPreset {
	if u.CurrentCharacterID >= 0 && u.CurrentCharacterID < len(u.Characters) {
		return u.Characters[u.CurrentCharacterID]
	}
	// Fallback to the first character or create a new one if somehow invalid
	if len(u.Characters) == 0 {
		u.Characters = []*CharacterPreset{NewCharacterPreset()}
		u.CurrentCharacterID = 0
	}
	return u.Characters[0]
}

// ChangeCurrentCharacter устанавливает текущего персонажа по индексу.
func (u *User) ChangeCurrentCharacter(index int) {
	if index >= 0 && index < len(u.Characters) {
		u.CurrentCharacterID = index
	}
}

// EnsureChatHistoryLimit обрезает историю чата, если она превышает лимит.
func (u *User) EnsureChatHistoryLimit(charIndex int, limit int) {
	if charIndex >= 0 && charIndex < len(u.Characters) {
		char := u.Characters[charIndex]
		if len(char.Chat) > limit {
			char.Chat = char.Chat[len(char.Chat)-limit:] // Оставляем только 'limit' последних сообщений
		}
	}
}

// ReplacePlaceholders replaces {{user}} and {{char}} placeholders in a string.
func (u *User) ReplacePlaceholders(input string) string {
	input = strings.ReplaceAll(input, "{{user}}", u.UserName)
	input = strings.ReplaceAll(input, "{{char}}", u.GetCurrentCharacter().Name)
	return input
}
