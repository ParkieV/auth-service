package usecase

import (
	"context"
	"errors"
	"github.com/ParkieV/auth-service/internal/infrastructure/auth_client"
	"log/slog"
)

var (
	ErrTokenInvalid = errors.New("token invalid or expired")
)

type VerifyUsecase struct {
	ac  auth_client.AuthClient
	log *slog.Logger
}

func NewVerifyUsecase(ac auth_client.AuthClient, log *slog.Logger) *VerifyUsecase {
	return &VerifyUsecase{ac: ac, log: log}
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
	return &VerifyResult{
		UserID: uid,
		Active: true,
	}, nil
}
