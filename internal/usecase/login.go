package usecase

import (
	"errors"
	"time"

	"github.com/ParkieV/auth-service/internal/domain"
)

// ErrInvalidCredentials возвращается, если Keycloak отверг credentials.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrNotConfirmed возвращается, если e-mail ещё не подтверждён.
var ErrNotConfirmed = errors.New("email not confirmed")

// ErrUserNotFound возвращается, если пользователь не найден.
var ErrUserNotFound = errors.New("user not found")

// LoginUsecase — юзкейc логина.
type LoginUsecase struct {
	repo  UserRepository
	kc    KeycloakClient
	cache Cache
}

// NewLoginUsecase создаёт LoginUsecase.
func NewLoginUsecase(repo UserRepository, kc KeycloakClient, cache Cache) *LoginUsecase {
	return &LoginUsecase{repo: repo, kc: kc, cache: cache}
}

// Login проверяет credentials, возвращает пару токенов или ошибку.
func (uc *LoginUsecase) Login(emailStr, password string) (string, string, error) {
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		return "", "", err
	}

	user, err := uc.repo.FindByEmail(email)
	if err != nil {
		return "", "", ErrUserNotFound
	}
	if !user.Confirmed {
		return "", "", ErrNotConfirmed
	}

	access, refresh, err := uc.kc.Authenticate(email.String(), password)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	// кешируем refresh→userID на 24 часа
	_ = uc.cache.Set(refresh, user.ID, 24*time.Hour)
	return access, refresh, nil
}
