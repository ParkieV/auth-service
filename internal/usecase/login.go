package usecase

import (
	"errors"
	"time"

	"github.com/ParkieV/auth-service/internal/domain"
)

var (
	// ErrUserNotFound возвращается, если пользователь не найден
	ErrUserNotFound = errors.New("user not found")
	// ErrNotConfirmed — e-mail ещё не подтверждён
	ErrNotConfirmed = errors.New("email not confirmed")
	// ErrInvalidCredentials — неверный логин/пароль
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// KeycloakClient отвечает за аутентификацию через Keycloak
type KeycloakClient interface {
	Authenticate(email, password string) (accessToken, refreshToken string, err error)
}

// Cache отвечает за кеширование (например, usecase-token → userID)
type Cache interface {
	Set(key, value string, ttl time.Duration) error
}

// LoginUsecase — юзкейc логина
type LoginUsecase struct {
	repo  UserRepository
	kc    KeycloakClient
	cache Cache
}

// NewLoginUsecase создаёт LoginUsecase
func NewLoginUsecase(repo UserRepository, kc KeycloakClient, cache Cache) *LoginUsecase {
	return &LoginUsecase{repo: repo, kc: kc, cache: cache}
}

// Login проверяет credentials и возвращает пару токенов или ошибку
func (uc *LoginUsecase) Login(emailStr, password string) (string, string, error) {
	// 1) Валидация e-mail
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		return "", "", err
	}

	// 2) Забираем пользователя
	user, err := uc.repo.FindByEmail(email)
	if err != nil {
		return "", "", ErrUserNotFound
	}

	// 3) Проверяем, что e-mail подтверждён
	if !user.Confirmed {
		return "", "", ErrNotConfirmed
	}

	// 4) Аутентификация в Keycloak
	access, refresh, err := uc.kc.Authenticate(email.String(), password)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	// 5) Опционально кешируем usecase-token для возможного отзыва
	_ = uc.cache.Set(refresh, user.ID, 24*time.Hour)

	return access, refresh, nil
}
