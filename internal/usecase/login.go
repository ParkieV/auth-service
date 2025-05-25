package usecase

import (
	"bytes"
	"errors"
	"time"

	"github.com/ParkieV/auth-service/internal/domain"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

var ErrNotConfirmed = errors.New("email not confirmed")

var ErrUserNotFound = errors.New("user not found")

type LoginUsecase struct {
	repo  UserRepository
	kc    KeycloakClient
	cache Cache
}

func NewLoginUsecase(repo UserRepository, kc KeycloakClient, cache Cache) *LoginUsecase {
	return &LoginUsecase{repo: repo, kc: kc, cache: cache}
}

func (uc *LoginUsecase) Login(emailStr string, password bytes.Buffer) (string, string, error) {
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

	_ = uc.cache.Set(refresh, user.ID, 24*time.Hour)
	return access, refresh, nil
}
