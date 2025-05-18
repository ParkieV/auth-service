package usecase_tests

import (
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/ParkieV/auth-service/internal/domain"
)

// Мок для UserRepository — и Save, и FindByEmail
type MockUserRepo struct{ mock.Mock }

func (m *MockUserRepo) Save(u *domain.User) error {
	return m.Called(u).Error(0)
}

func (m *MockUserRepo) FindByEmail(email domain.Email) (*domain.User, error) {
	args := m.Called(email)
	if u := args.Get(0); u != nil {
		return u.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

// Мок для MessageBroker
type MockBroker struct{ mock.Mock }

func (m *MockBroker) Publish(queue string, body []byte) error {
	return m.Called(queue, body).Error(0)
}

// Мок для KeycloakClient
type MockKC struct{ mock.Mock }

func (m *MockKC) Authenticate(email, password string) (string, string, error) {
	args := m.Called(email, password)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockKC) RefreshToken(refreshToken string) (string, string, error) {
	args := m.Called(refreshToken)
	return args.String(0), args.String(1), args.Error(2)
}

// Мок для Cache
type MockCache struct{ mock.Mock }

func (m *MockCache) Set(key, value string, ttl time.Duration) error {
	return m.Called(key, value, ttl).Error(0)
}

func (m *MockCache) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}

func (m *MockCache) Delete(key string) error {
	return m.Called(key).Error(0)
}
