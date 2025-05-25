package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/ParkieV/auth-service/internal/domain"
)

type UserRepository interface {
	Save(u *domain.User) error
	FindByEmail(email domain.Email) (*domain.User, error)
}

type MessageBroker interface {
	Publish(queue string, body []byte) error
}

type RegisterUsecase struct {
	repo   UserRepository
	broker MessageBroker
	ttl    time.Duration
}

func NewRegisterUsecase(repo UserRepository, broker MessageBroker, confirmationTTL time.Duration) *RegisterUsecase {
	return &RegisterUsecase{repo: repo, broker: broker, ttl: confirmationTTL}
}

func (uc *RegisterUsecase) Register(emailStr, passwordHash string) (string, error) {
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		return "", err
	}

	userID := uuid.NewString()
	confirmID := generateRandomID()
	expiresAt := time.Now().Add(uc.ttl)

	user := domain.NewUser(userID, email, passwordHash, confirmID, expiresAt)

	if err := uc.repo.Save(user); err != nil {
		return "", err
	}

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
