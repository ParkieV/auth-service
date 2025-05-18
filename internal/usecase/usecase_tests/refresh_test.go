package usecase_tests

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ParkieV/auth-service/internal/usecase"
)

func TestRefresh_Success(t *testing.T) {
	kc := &MockKC{}
	cache := &MockCache{}
	ttl := 72 * time.Hour
	uc := usecase.NewRefreshUsecase(kc, cache, ttl)

	cache.On("Get", "old-refresh").Return("user-123", nil)
	kc.On("RefreshToken", "old-refresh").Return("new-access", "new-refresh", nil)
	cache.On("Set", "new-refresh", "user-123", ttl).Return(nil)
	cache.On("Delete", "old-refresh").Return(nil)

	access, refresh, err := uc.Refresh("old-refresh")
	assert.NoError(t, err)
	assert.Equal(t, "new-access", access)
	assert.Equal(t, "new-refresh", refresh)
}

func TestRefresh_InvalidToken(t *testing.T) {
	cache := &MockCache{}
	uc := usecase.NewRefreshUsecase(&MockKC{}, cache, time.Hour)

	cache.On("Get", "bad-token").Return("", errors.New("not found"))

	_, _, err := uc.Refresh("bad-token")
	assert.ErrorIs(t, err, usecase.ErrInvalidRefreshToken)
}

func TestRefresh_KeycloakError(t *testing.T) {
	kc := &MockKC{}
	cache := &MockCache{}
	uc := usecase.NewRefreshUsecase(kc, cache, time.Hour)

	cache.On("Get", "refresh").Return("user-1", nil)
	kc.On("RefreshToken", "refresh").Return("", "", errors.New("kc down"))

	_, _, err := uc.Refresh("refresh")
	assert.ErrorIs(t, err, usecase.ErrRefreshFailed)
}
