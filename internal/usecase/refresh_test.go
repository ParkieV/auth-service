package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ParkieV/auth-service/internal/usecase"
)

type mockKCRefresh struct{ mock.Mock }

func (m *mockKCRefresh) Authenticate(email, password string) (string, string, error) {
	return "", "", nil
}
func (m *mockKCRefresh) RefreshToken(refreshToken string) (string, string, error) {
	args := m.Called(refreshToken)
	return args.String(0), args.String(1), args.Error(2)
}

type mockCacheRefresh struct{ mock.Mock }

func (m *mockCacheRefresh) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}
func (m *mockCacheRefresh) Set(key, value string, ttl time.Duration) error {
	return m.Called(key, value, ttl).Error(0)
}
func (m *mockCacheRefresh) Delete(key string) error {
	return m.Called(key).Error(0)
}

func TestRefresh_Success(t *testing.T) {
	kc := &mockKCRefresh{}
	cache := &mockCacheRefresh{}
	ttl := 72 * time.Hour
	uc := usecase.NewRefreshUsecase(kc, cache, ttl)

	// Сценарий: токен найден в кеше → Keycloak обновил → кеш обновился → старый удалён
	cache.On("Get", "old-usecase").Return("user-123", nil)
	kc.On("RefreshToken", "old-usecase").Return("new-access", "new-usecase", nil)
	cache.On("Set", "new-usecase", "user-123", ttl).Return(nil)
	cache.On("Delete", "old-usecase").Return(nil)

	access, refresh, err := uc.Refresh("old-usecase")
	assert.NoError(t, err)
	assert.Equal(t, "new-access", access)
	assert.Equal(t, "new-usecase", refresh)

	cache.AssertCalled(t, "Get", "old-usecase")
	kc.AssertCalled(t, "RefreshToken", "old-usecase")
	cache.AssertCalled(t, "Set", "new-usecase", "user-123", ttl)
	cache.AssertCalled(t, "Delete", "old-usecase")
}

func TestRefresh_InvalidToken(t *testing.T) {
	uc := usecase.NewRefreshUsecase(&mockKCRefresh{}, &mockCacheRefresh{}, time.Hour)
	cache := uc.cache.(*mockCacheRefresh)

	cache.On("Get", "bad-token").Return("", errors.New("not found"))

	_, _, err := uc.Refresh("bad-token")
	assert.ErrorIs(t, err, usecase.ErrInvalidRefreshToken)

	cache.AssertCalled(t, "Get", "bad-token")
}

func TestRefresh_KeycloakError(t *testing.T) {
	kc := &mockKCRefresh{}
	cache := &mockCacheRefresh{}
	uc := usecase.NewRefreshUsecase(kc, cache, time.Hour)

	cache.On("Get", "usecase").Return("user-1", nil)
	kc.On("RefreshToken", "usecase").Return("", "", errors.New("kc down"))

	_, _, err := uc.Refresh("usecase")
	assert.ErrorIs(t, err, usecase.ErrRefreshFailed)

	cache.AssertCalled(t, "Get", "usecase")
	kc.AssertCalled(t, "RefreshToken", "usecase")
}
