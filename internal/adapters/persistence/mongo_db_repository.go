package persistence

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/alex-pyslar/neuro-chat-bot/internal/domain"
	"github.com/alex-pyslar/neuro-chat-bot/internal/usecases"
	"github.com/alex-pyslar/neuro-chat-bot/pkg/logger"
)

// MongoDbRepository является реализацией usecases.UserRepository для MongoDB.
type MongoDbRepository struct {
	usersCollection *mongo.Collection
	logger          logger.Logger
}

// NewMongoDbRepository создает новый экземпляр MongoDbRepository.
func NewMongoDbRepository(connectionString, databaseName string, logger logger.Logger) (*MongoDbRepository, error) {
	clientOptions := options.Client().ApplyURI(connectionString)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.Error("Failed to connect to MongoDB: %v", err)
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Проверяем соединение
	err = client.Ping(ctx, nil)
	if err != nil {
		logger.Error("Failed to ping MongoDB: %v", err)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	logger.Info("Connected to MongoDB!")

	usersCollection := client.Database(databaseName).Collection("users")

	return &MongoDbRepository{
		usersCollection: usersCollection,
		logger:          logger,
	}, nil
}

// SaveUser сохраняет или обновляет пользователя в базе данных.
func (r *MongoDbRepository) SaveUser(ctx context.Context, user *domain.User) error {
	opts := options.Update().SetUpsert(true)
	filter := bson.M{"_id": user.ID}
	update := bson.M{"$set": user} // Используем $set для полного обновления документа

	_, err := r.usersCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		r.logger.Error("Error saving user %d: %v", user.ID, err)
		return fmt.Errorf("error saving user %d: %w", user.ID, err)
	}
	return nil
}

// LoadUser загружает пользователя по ID.
func (r *MongoDbRepository) LoadUser(ctx context.Context, userID int64) (*domain.User, error) {
	filter := bson.M{"_id": userID}
	var user domain.User
	err := r.usersCollection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Пользователь не найден
		}
		r.logger.Error("Error loading user %d: %v", userID, err)
		return nil, fmt.Errorf("error loading user %d: %w", userID, err)
	}
	return &user, nil
}

// AddChatMessage добавляет сообщение чата для указанного пользователя и персонажа.
func (r *MongoDbRepository) AddChatMessage(ctx context.Context, userID int64, characterIndex int, message domain.ChatMessage) error {
	filter := bson.M{"_id": userID}
	update := bson.M{"$push": bson.M{fmt.Sprintf("characters.%d.chat", characterIndex): message}}

	result, err := r.usersCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		r.logger.Error("Error adding chat message for user %d, character index %d: %v", userID, characterIndex, err)
		return fmt.Errorf("error adding chat message for user %d, character index %d: %w", userID, characterIndex, err)
	}

	if result.MatchedCount == 0 {
		r.logger.Error("User %d not found when trying to add chat message.", userID)
		return fmt.Errorf("user %d not found when trying to add chat message", userID)
	}
	return nil
}

// EnsureChatHistoryLimit обрезает историю чата для указанного пользователя и персонажа.
func (r *MongoDbRepository) EnsureChatHistoryLimit(ctx context.Context, userID int64, characterIndex int, limit int) error {
	// This operation is more complex with MongoDB's $slice.
	// It's often easier to manage chat history limits in the application layer (UserInteractor)
	// by fetching the user, trimming the chat slice, and then saving the user.
	// Direct MongoDB $slice for updates that keep only the last N elements is tricky if N is dynamic.
	// If performance is an issue for very long chat histories, consider a separate collection for chat messages.

	// For now, we'll rely on the application layer to manage the limit before saving the user.
	// If you want a database-level enforcement, you'd typically pull, trim, and push.
	// Or use aggregations or more complex updates.
	// This method might be left empty or removed if the use case ensures the limit before SaveUser.
	r.logger.Warn("EnsureChatHistoryLimit is not implemented for direct MongoDB operation. Handled by application layer.")
	return nil
}

// Verify that MongoDbRepository implements usecases.UserRepository
var _ usecases.UserRepository = (*MongoDbRepository)(nil)
