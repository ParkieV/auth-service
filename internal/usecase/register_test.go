package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ParkieV/auth-service/internal/domain"
)

type mockRepo struct{ mock.Mock }

func (m *mockRepo) Save(u *domain.User) error {
	return m.Called(u).Error(0)
}

type mockBroker struct{ mock.Mock }

func (m *mockBroker) Publish(queue string, body []byte) error {
	return m.Called(queue, body).Error(0)
}

func TestRegister_Success(t *testing.T) {
	repo := &mockRepo{}
	broker := &mockBroker{}
	ttl := time.Hour * 24
	uc := refresh.NewRegisterUsecase(repo, broker, ttl)

	repo.
		On("Save", mock.AnythingOfType("*domain.User")).
		Return(nil)
	broker.
		On("Publish", "email.confirm", mock.Anything).
		Return(nil)

	id, err := uc.Register("alice@example.com", "hashedpwd")
	assert.NoError(t, err)
	assert.NotEmpty(t, id)

	repo.AssertCalled(t, "Save", mock.AnythingOfType("*domain.User"))
	broker.AssertCalled(t, "Publish", "email.confirm", mock.Anything)
}

func TestRegister_InvalidEmail(t *testing.T) {
	uc := refresh.NewRegisterUsecase(nil, nil, time.Hour)
	_, err := uc.Register("not-an-email", "pwd")
	assert.ErrorIs(t, err, domain.ErrInvalidEmail)
}

func TestRegister_RepoError(t *testing.T) {
	repo := &mockRepo{}
	broker := &mockBroker{}
	uc := refresh.NewRegisterUsecase(repo, broker, time.Hour)

	repo.
		On("Save", mock.Anything).
		Return(errors.New("db failure"))

	_, err := uc.Register("bob@example.com", "pwd")
	assert.EqualError(t, err, "db failure")
}

func TestRegister_BrokerError(t *testing.T) {
	repo := &mockRepo{}
	broker := &mockBroker{}
	uc := refresh.NewRegisterUsecase(repo, broker, time.Hour)

	repo.
		On("Save", mock.Anything).
		Return(nil)
	broker.
		On("Publish", "email.confirm", mock.Anything).
		Return(errors.New("broker down"))

	_, err := uc.Register("eve@example.com", "pwd")
	assert.EqualError(t, err, "broker down")
}
