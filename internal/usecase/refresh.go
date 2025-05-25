package usecase

import (
	"errors"
	"time"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid usecase token")
	ErrRefreshFailed       = errors.New("usecase failed")
)

type KeycloakClient interface {
	Authenticate(email, password string) (accessToken, refreshToken string, err error)
	RefreshToken(refreshToken string) (newAccessToken, newRefreshToken string, err error)
}

type Cache interface {
	Get(key string) (string, error)
	Set(key, value string, ttl time.Duration) error
	Delete(key string) error
}

type RefreshUsecase struct {
	kc         KeycloakClient
	cache      Cache
	refreshTTL time.Duration
}

func NewRefreshUsecase(kc KeycloakClient, cache Cache, refreshTTL time.Duration) *RefreshUsecase {
	return &RefreshUsecase{kc: kc, cache: cache, refreshTTL: refreshTTL}
}

func (uc *RefreshUsecase) Refresh(refreshToken string) (string, string, error) {
	userID, err := uc.cache.Get(refreshToken)
	if err != nil {
		return "", "", ErrInvalidRefreshToken
	}

	newAccess, newRefresh, err := uc.kc.RefreshToken(refreshToken)
	if err != nil {
		return "", "", ErrRefreshFailed
	}

	if err := uc.cache.Set(newRefresh, userID, uc.refreshTTL); err != nil {
	}

	_ = uc.cache.Delete(refreshToken)

	return newAccess, newRefresh, nil
}
