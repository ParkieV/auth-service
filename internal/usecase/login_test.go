package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ParkieV/auth-service/internal/domain"
)

type mockKC struct{ mock.Mock }

func (m *mockKC) Authenticate(email, password string) (string, string, error) {
	args := m.Called(email, password)
	return args.String(0), args.String(1), args.Error(2)
}

type mockCache struct{ mock.Mock }

func (m *mockCache) Set(key, value string, ttl time.Duration) error {
	return m.Called(key, value, ttl).Error(0)
}

type mockRepoLogin struct{ mock.Mock }

func (m *mockRepoLogin) FindByEmail(email domain.Email) (*domain.User, error) {
	args := m.Called(email)
	if u := args.Get(0); u != nil {
		return u.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestLogin_Success(t *testing.T) {
	repo := &mockRepoLogin{}
	kc := &mockKC{}
	cache := &mockCache{}
	uc := refresh.NewLoginUsecase(repo, kc, cache)

	emailVO, _ := domain.NewEmail("alice@example.com")
	user := domain.NewUser("uid-1", emailVO, "hash", "code", time.Now().Add(time.Hour))
	user.Confirmed = true

	repo.On("FindByEmail", emailVO).Return(user, nil)
	kc.On("Authenticate", emailVO.String(), "pass").Return("tok", "ref", nil)
	cache.On("Set", "ref", "uid-1", mock.Anything).Return(nil)

	access, refresh, err := uc.Login("alice@example.com", "pass")
	assert.NoError(t, err)
	assert.Equal(t, "tok", access)
	assert.Equal(t, "ref", refresh)

	repo.AssertCalled(t, "FindByEmail", emailVO)
	kc.AssertCalled(t, "Authenticate", emailVO.String(), "pass")
	cache.AssertCalled(t, "Set", "ref", "uid-1", mock.Anything)
}

func TestLogin_InvalidEmail(t *testing.T) {
	uc := refresh.NewLoginUsecase(nil, nil, nil)
	_, _, err := uc.Login("bad-email", "pwd")
	assert.ErrorIs(t, err, domain.ErrInvalidEmail)
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockRepoLogin{}
	uc := refresh.NewLoginUsecase(repo, &mockKC{}, &mockCache{})

	emailVO, _ := domain.NewEmail("bob@example.com")
	repo.On("FindByEmail", emailVO).Return(nil, errors.New("no rows"))

	_, _, err := uc.Login("bob@example.com", "pwd")
	assert.ErrorIs(t, err, refresh.ErrUserNotFound)
}

func TestLogin_NotConfirmed(t *testing.T) {
	repo := &mockRepoLogin{}
	uc := refresh.NewLoginUsecase(repo, &mockKC{}, &mockCache{})

	emailVO, _ := domain.NewEmail("eve@example.com")
	user := domain.NewUser("uid-2", emailVO, "hash", "code", time.Now().Add(time.Hour))
	// user.Confirmed == false

	repo.On("FindByEmail", emailVO).Return(user, nil)

	_, _, err := uc.Login("eve@example.com", "pwd")
	assert.ErrorIs(t, err, refresh.ErrNotConfirmed)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	repo := &mockRepoLogin{}
	kc := &mockKC{}
	uc := refresh.NewLoginUsecase(repo, kc, &mockCache{})

	emailVO, _ := domain.NewEmail("alice@example.com")
	user := domain.NewUser("uid-3", emailVO, "hash", "code", time.Now().Add(time.Hour))
	user.Confirmed = true

	repo.On("FindByEmail", emailVO).Return(user, nil)
	kc.On("Authenticate", emailVO.String(), "wrong").Return("", "", errors.New("denied"))

	_, _, err := uc.Login("alice@example.com", "wrong")
	assert.ErrorIs(t, err, refresh.ErrInvalidCredentials)
}
