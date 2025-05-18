package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/ParkieV/auth-service/internal/domain"
)

// UserRepository отвечает за работу с users (save & fetch).
type UserRepository interface {
	Save(u *domain.User) error
	FindByEmail(email domain.Email) (*domain.User, error)
}

// MessageBroker отвечает за публикацию событий в очереди.
type MessageBroker interface {
	Publish(queue string, body []byte) error
}

// RegisterUsecase — юзкейc регистрации.
type RegisterUsecase struct {
	repo   UserRepository
	broker MessageBroker
	ttl    time.Duration
}

// NewRegisterUsecase создаёт RegisterUsecase с заданным TTL для confirmation-кода.
func NewRegisterUsecase(repo UserRepository, broker MessageBroker, confirmationTTL time.Duration) *RegisterUsecase {
	return &RegisterUsecase{repo: repo, broker: broker, ttl: confirmationTTL}
}

// Register создаёт нового пользователя и публикует событие на e-mail.
func (uc *RegisterUsecase) Register(emailStr, passwordHash string) (string, error) {
	// валидация e-mail
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		return "", err
	}

	// генерируем ID и код подтверждения
	userID := uuid.NewString()
	confirmID := generateRandomID()
	expiresAt := time.Now().Add(uc.ttl)

	// собираем доменный объект
	user := domain.NewUser(userID, email, passwordHash, confirmID, expiresAt)

	// сохраняем
	if err := uc.repo.Save(user); err != nil {
		return "", err
	}

	// публикуем событие для отправки письма
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

func generateRandomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
