package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ParkieV/auth-service/internal/infrastructure/auth_client"
	"github.com/ParkieV/auth-service/internal/infrastructure/broker"
	"github.com/ParkieV/auth-service/internal/infrastructure/db"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ParkieV/auth-service/internal/domain"
)

var (
	ErrEmailExists = errors.New("email already exists")
)

type RegisterUsecase struct {
	repo   db.UserMutRepository
	broker broker.MessageBroker
	ac     auth_client.AuthClient
	ttl    time.Duration
	log    *slog.Logger
}

func NewRegisterUsecase(repo db.UserMutRepository, broker broker.MessageBroker, ac auth_client.AuthClient, confirmationTTL time.Duration, log *slog.Logger) *RegisterUsecase {
	return &RegisterUsecase{repo: repo, broker: broker, ac: ac, ttl: confirmationTTL, log: log}
}

func (uc *RegisterUsecase) Register(ctx context.Context, emailStr, plainPassword string) (string, error) {
	email, err := domain.NewEmail(strings.TrimSpace(emailStr))
	if err != nil {
		uc.log.Info("invalid email", "email", emailStr, "err", err)
		return "", err
	}

	userID := uuid.NewString()
	confirmID := uuid.NewString()
	user, err := domain.NewUserFromRegistration(
		userID, email, plainPassword, confirmID, uc.ttl,
	)
	if err != nil {
		uc.log.Error("build user failed", "err", err)
		return "", err
	}

	if err := uc.repo.Save(ctx, user); err != nil {
		switch {
		case ctx.Err() != nil:
			return "", ctx.Err()
		case errors.Is(err, db.ErrDuplicateKey):
			return "", ErrEmailExists
		default:
			uc.log.Error("save user failed", "err", err)
			return "", err
		}
	}

	msg := struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
		Code   string `json:"code"`
	}{
		UserID: userID,
		Email:  email.String(),
		Code:   confirmID,
	}
	body, err := json.Marshal(msg)
	if err != nil {
		uc.log.Error("marshal confirm payload failed", "err", err)
	}

	if err := uc.broker.PublishToQueue(ctx, "email.confirm", body); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		uc.log.Error("publish confirm email failed", "err", err)
	}

	msg2 := struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}{
		UserID: userID,
		Email:  email.String(),
	}

	body, err = json.Marshal(msg2)
	if err != nil {
		uc.log.Error("marshal confirm payload failed", "err", err)
	}

	if err := uc.broker.PublishToTopic(ctx, "UserRegistered", body); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		uc.log.Error("publish confirm email failed", "err", err)
	}

	return userID, nil
}
