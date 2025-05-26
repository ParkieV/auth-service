package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ParkieV/auth-service/internal/infrastructure/auth_client"
	"github.com/ParkieV/auth-service/internal/infrastructure/broker"
	"log/slog"
)

var (
	ErrTokenInvalid = errors.New("token invalid or expired")
)

type VerifyUsecase struct {
	ac     auth_client.AuthClient
	broker broker.MessageBroker
	log    *slog.Logger
}

func NewVerifyUsecase(ac auth_client.AuthClient, broker broker.MessageBroker, log *slog.Logger) *VerifyUsecase {
	return &VerifyUsecase{ac: ac, broker: broker, log: log}
}

type VerifyResult struct {
	UserID string
	Scope  []string
	Active bool
}

func (uc *VerifyUsecase) Verify(ctx context.Context, token string) (*VerifyResult, error) {
	active, uid, err := uc.ac.VerifyAccess(ctx, token)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		uc.log.Error("verify access failed", "err", err)
		return nil, err
	}
	if !active {
		return nil, ErrTokenInvalid
	}

	msg := struct {
		UserID string `json:"user_id"`
		Active bool   `json:"active"`
	}{
		UserID: uid,
		Active: true,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		uc.log.Error("marshal confirm payload failed", "err", err)
	}

	if err := uc.broker.PublishToTopic(ctx, "UserLoggedIn", body); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		uc.log.Error("publish confirm email failed", "err", err)
	}

	return &VerifyResult{
		UserID: uid,
		Active: true,
	}, nil
}
