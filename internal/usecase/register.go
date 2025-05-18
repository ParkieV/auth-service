package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/ParkieV/auth-service/internal/domain"
)

// UserRepository отвечает за сохранение User
type UserRepository interface {
	Save(u *domain.User) error
}

// MessageBroker отвечает за публикацию сообщений
type MessageBroker interface {
	Publish(queue string, body []byte) error
}

// RegisterUsecase — юзкейc регистрации
type RegisterUsecase struct {
	repo            UserRepository
	broker          MessageBroker
	confirmationTTL time.Duration
}

// NewRegisterUsecase создаёт RegisterUsecase.
// confirmationTTL — время жизни кода подтверждения.
func NewRegisterUsecase(repo UserRepository, broker MessageBroker, confirmationTTL time.Duration) *RegisterUsecase {
	return &RegisterUsecase{
		repo:            repo,
		broker:          broker,
		confirmationTTL: confirmationTTL,
	}
}

// Register создаёт нового пользователя и публикует событие для отправки e-mail.
// Возвращает ID созданного пользователя или ошибку.
func (uc *RegisterUsecase) Register(emailStr, passwordHash string) (string, error) {
	// 1) Валидация e-mail
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		return "", err
	}

	// 2) Генерация ID пользователя и confirmation code
	userID := uuid.NewString()
	confirmID := generateRandomID()
	expiresAt := time.Now().Add(uc.confirmationTTL)

	// 3) Конструируем доменный объект
	user := domain.NewUser(userID, email, passwordHash, confirmID, expiresAt)

	// 4) Сохраняем в БД
	if err := uc.repo.Save(user); err != nil {
		return "", err
	}

	// 5) Публикуем сообщение для отправки письма
	payload := struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}{
		Email: email.String(),
		Code:  confirmID,
	}
	body, _ := json.Marshal(payload)

	if err := uc.broker.Publish("email.confirm", body); err != nil {
		return "", err
	}

	return userID, nil
}

// generateRandomID — вспомогательная функция для confirmation code
func generateRandomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
