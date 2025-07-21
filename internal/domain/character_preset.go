package domain

import (
	"strings"
)

// CharacterPreset содержит настройки для конкретного персонажа.
type CharacterPreset struct {
	ID       int           `json:"id" bson:"id"`             // ID персонажа, например, для выбора из списка
	Name     string        `json:"name" bson:"name"`         // Имя персонажа
	Greeting string        `json:"greeting" bson:"greeting"` // Приветствие персонажа
	Prompt   string        `json:"prompt" bson:"prompt"`     // Системный промпт для персонажа
	Chat     []ChatMessage `json:"chat" bson:"chat"`         // История чата с этим персонажем
}

// NewCharacterPreset создает новый экземпляр CharacterPreset с настройками по умолчанию.
func NewCharacterPreset() *CharacterPreset {
	return &CharacterPreset{
		ID:       0, // Будет автоматически назначен при добавлении в список
		Name:     "Default",
		Greeting: "Hello! How can I help you today?",
		Prompt:   "You are a helpful AI assistant.",
		Chat:     []ChatMessage{},
	}
}

// GetChatMessagesForModel возвращает историю чата в формате, подходящем для модели.
func (cp *CharacterPreset) GetChatMessagesForModel() []ChatMessage {
	var messages []ChatMessage

	// Добавляем системный промпт
	if cp.Prompt != "" {
		messages = append(messages, NewChatMessage(System, cp.Prompt))
	}

	// Добавляем историю чата
	messages = append(messages, cp.Chat...)

	return messages
}

// ReplacePlaceholders replaces {{char}} placeholder in a string.
func (cp *CharacterPreset) ReplacePlaceholders(input string) string {
	return strings.ReplaceAll(input, "{{char}}", cp.Name)
}
