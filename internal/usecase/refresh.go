package usecase

import (
	"errors"
	"time"
)

// Ошибки модуля Refresh
var (
	// ErrInvalidRefreshToken возвращается, если токен отсутствует в кеше
	ErrInvalidRefreshToken = errors.New("invalid usecase token")
	// ErrRefreshFailed — сбой при обновлении у Keycloak
	ErrRefreshFailed = errors.New("usecase failed")
)

// KeycloakClient отвечает теперь не только за Authenticate, но и за RefreshToken
type KeycloakClient interface {
	Authenticate(email, password string) (accessToken, refreshToken string, err error)
	RefreshToken(refreshToken string) (newAccessToken, newRefreshToken string, err error)
}

// Cache отвечает за кеширование (refreshToken → userID)
type Cache interface {
	Get(key string) (string, error)
	Set(key, value string, ttl time.Duration) error
	Delete(key string) error
}

// RefreshUsecase — юзкейс обновления токенов
type RefreshUsecase struct {
	kc         KeycloakClient
	cache      Cache
	refreshTTL time.Duration
}

// NewRefreshUsecase создаёт RefreshUsecase
func NewRefreshUsecase(kc KeycloakClient, cache Cache, refreshTTL time.Duration) *RefreshUsecase {
	return &RefreshUsecase{kc: kc, cache: cache, refreshTTL: refreshTTL}
}

// Refresh проверяет переданный refreshToken в кеше, запрашивает у Keycloak новую пару токенов,
// обновляет кеш и возвращает новые токены.
func (uc *RefreshUsecase) Refresh(refreshToken string) (string, string, error) {
	// 1) Проверяем, есть ли такой refreshToken в кеше
	userID, err := uc.cache.Get(refreshToken)
	if err != nil {
		return "", "", ErrInvalidRefreshToken
	}

	// 2) Просим Keycloak обновить токены
	newAccess, newRefresh, err := uc.kc.RefreshToken(refreshToken)
	if err != nil {
		return "", "", ErrRefreshFailed
	}

	// 3) В кеш записываем новый refreshToken
	if err := uc.cache.Set(newRefresh, userID, uc.refreshTTL); err != nil {
		// не фатально, но логировать стоит
	}

	// 4) Удаляем старый токен из кеша
	_ = uc.cache.Delete(refreshToken)

	return newAccess, newRefresh, nil
}
