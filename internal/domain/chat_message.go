package domain

// RoleEnums определяет роли в чате.
type RoleEnums int

const (
	System RoleEnums = iota
	Assistant
	UserRole
)

// String возвращает строковое представление роли.
func (r RoleEnums) String() string {
	switch r {
	case System:
		return "system"
	case Assistant:
		return "assistant"
	case UserRole:
		return "user"
	default:
		// В случае, если по какой-то причине роль не определена,
		// вернем "user" как безопасное значение по умолчанию, хотя лучше
		// гарантировать, что такого не произойдет.
		return "user"
	}
}

// ChatMessage представляет отдельное сообщение в чате.
type ChatMessage struct {
	// ERole больше не нужен для сохранения/JSON, так как Role будет строкой.
	// Оставляем для совместимости NewChatMessage, но он больше не будет сохраняться в БД.
	ERole   RoleEnums `json:"-" bson:"-"`       // Игнорируем ERole для JSON и BSON
	Role    string    `json:"role" bson:"role"` // Теперь Role (строка) сохраняется в DB и используется для JSON
	Content string    `json:"content" bson:"content"`
}

// NewChatMessage создает новое сообщение чата.
func NewChatMessage(role RoleEnums, content string) ChatMessage {
	return ChatMessage{
		ERole:   role,          // Сохраняем для внутреннего использования, если потребуется
		Role:    role.String(), // Устанавливаем строковое представление
		Content: content,
	}
}
