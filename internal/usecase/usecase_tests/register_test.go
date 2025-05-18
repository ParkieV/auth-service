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

func TestRegister_Success(t *testing.T) {
	repo := &MockUserRepo{}
	broker := &MockBroker{}
	uc := usecase.NewRegisterUsecase(repo, broker, 24*time.Hour)

	repo.On("Save", mock.AnythingOfType("*domain.User")).Return(nil)
	broker.On("Publish", "email.confirm", mock.Anything).Return(nil)

	id, err := uc.Register("alice@example.com", "hashpwd")
	assert.NoError(t, err)
	assert.NotEmpty(t, id)

	repo.AssertCalled(t, "Save", mock.Anything)
	broker.AssertCalled(t, "Publish", "email.confirm", mock.Anything)
}

func TestRegister_InvalidEmail(t *testing.T) {
	uc := usecase.NewRegisterUsecase(nil, nil, time.Hour)
	_, err := uc.Register("not-an-email", "pwd")
	assert.ErrorIs(t, err, domain.ErrInvalidEmail)
}

func TestRegister_RepoError(t *testing.T) {
	repo := &MockUserRepo{}
	broker := &MockBroker{}
	uc := usecase.NewRegisterUsecase(repo, broker, time.Hour)

	repo.On("Save", mock.Anything).Return(errors.New("db failure"))

	_, err := uc.Register("bob@example.com", "pwd")
	assert.EqualError(t, err, "db failure")
}

func TestRegister_BrokerError(t *testing.T) {
	repo := &MockUserRepo{}
	broker := &MockBroker{}
	uc := usecase.NewRegisterUsecase(repo, broker, time.Hour)

	repo.On("Save", mock.Anything).Return(nil)
	broker.On("Publish", "email.confirm", mock.Anything).Return(errors.New("mq down"))

	_, err := uc.Register("eve@example.com", "pwd")
	assert.EqualError(t, err, "mq down")
}
