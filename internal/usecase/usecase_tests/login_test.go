package usecase_tests

import (
	"errors"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ParkieV/auth-service/internal/domain"
	"github.com/ParkieV/auth-service/internal/usecase"
)

func TestLogin_Success(t *testing.T) {
	repo := &MockUserRepo{}
	kc := &MockKC{}
	cache := &MockCache{}
	uc := usecase.NewLoginUsecase(repo, kc, cache)

	emailVO, _ := domain.NewEmail("alice@example.com")
	user := domain.NewUser("uid", emailVO, "hash", "code", time.Now().Add(time.Hour))
	user.Confirmed = true

	repo.On("FindByEmail", emailVO).Return(user, nil)
	kc.On("Authenticate", emailVO.String(), "pass").Return("tok", "ref", nil)
	cache.On("Set", "ref", "uid", mock.Anything).Return(nil)

	access, refresh, err := uc.Login("alice@example.com", "pass")
	assert.NoError(t, err)
	assert.Equal(t, "tok", access)
	assert.Equal(t, "ref", refresh)
}

func TestLogin_InvalidEmail(t *testing.T) {
	uc := usecase.NewLoginUsecase(nil, nil, nil)
	_, _, err := uc.Login("bad-email", "pwd")
	assert.ErrorIs(t, err, domain.ErrInvalidEmail)
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &MockUserRepo{}
	uc := usecase.NewLoginUsecase(repo, &MockKC{}, &MockCache{})

	emailVO, _ := domain.NewEmail("bob@example.com")
	repo.On("FindByEmail", emailVO).Return(nil, errors.New("no rows"))

	_, _, err := uc.Login("bob@example.com", "pwd")
	assert.ErrorIs(t, err, usecase.ErrUserNotFound)
}

func TestLogin_NotConfirmed(t *testing.T) {
	repo := &MockUserRepo{}
	uc := usecase.NewLoginUsecase(repo, &MockKC{}, &MockCache{})

	emailVO, _ := domain.NewEmail("eve@example.com")
	user := domain.NewUser("uid2", emailVO, "h", "c", time.Now().Add(time.Hour))
	// user.Confirmed == false

	repo.On("FindByEmail", emailVO).Return(user, nil)

	_, _, err := uc.Login("eve@example.com", "pwd")
	assert.ErrorIs(t, err, usecase.ErrNotConfirmed)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	repo := &MockUserRepo{}
	kc := &MockKC{}
	uc := usecase.NewLoginUsecase(repo, kc, &MockCache{})

	emailVO, _ := domain.NewEmail("alice@example.com")
	user := domain.NewUser("uid3", emailVO, "h", "c", time.Now().Add(time.Hour))
	user.Confirmed = true

	repo.On("FindByEmail", emailVO).Return(user, nil)
	kc.On("Authenticate", emailVO.String(), "wrong").Return("", "", errors.New("denied"))

	_, _, err := uc.Login("alice@example.com", "wrong")
	assert.ErrorIs(t, err, usecase.ErrInvalidCredentials)
}
