package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ParkieV/auth-service/internal/infrastructure/auth_client"
	"github.com/ParkieV/auth-service/internal/infrastructure/broker"
	"github.com/ParkieV/auth-service/internal/infrastructure/cache"
	"github.com/ParkieV/auth-service/internal/infrastructure/db"
	"log/slog"
	"time"

	"github.com/ParkieV/auth-service/internal/domain"
)

var (
	ErrNotConfirmed       = errors.New("email not confirmed")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
)

type LoginUsecase struct {
	repo   db.UserMutRepository
	ac     auth_client.AuthClient
	cache  cache.Cache
	broker broker.MessageBroker
	log    *slog.Logger
}

func NewLoginUsecase(repo db.UserMutRepository, ac auth_client.AuthClient, cache cache.Cache, broker broker.MessageBroker, log *slog.Logger) *LoginUsecase {
	return &LoginUsecase{repo: repo, ac: ac, cache: cache, broker: broker, log: log}
}

func (uc *LoginUsecase) Login(ctx context.Context, emailStr, plainPassword string) (string, string, error) {
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		uc.log.Error("could not parse email", "err", err)
		return "", "", err
	}

	user, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		uc.log.Error("AUF1", "email", email)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", "", ctxErr
		}
		return "", "", ErrUserNotFound
	}
	//if !user.IsConfirmed() {
	//	return "", "", ErrNotConfirmed
	//}

	ok, needRehash := user.VerifyPassword(plainPassword)
	if !ok {
		return "", "", ErrInvalidCredentials
	}

	defer func() {
		if needRehash {
			if newPwd, _ := domain.NewPasswordFromPlain(plainPassword); newPwd.Hash() != user.HashForStorage() {
				if err := uc.repo.UpdatePasswordHash(ctx, user.ID(), newPwd.Hash()); err != nil {
					uc.log.WarnContext(ctx, "plainPassword rehash failed", "err", err)
				}
			}
		}
	}()

	access, refresh, err := uc.ac.GenerateTokens(ctx, user.ID())
	if err != nil {
		uc.log.Error("ERRORRR HERE", "error", err)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", "", ctxErr
		}
		return "", "", ErrInvalidCredentials
	}

	if err := uc.cache.Set(ctx, refresh, user.ID(), 24*time.Hour); err != nil {
		uc.log.WarnContext(ctx, "cache set failed", "err", err)
	}

	msg := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}{
		AccessToken:  access,
		RefreshToken: refresh,
	}
	body, err := json.Marshal(msg)
	if err != nil {
		uc.log.Error("marshal confirm payload failed", "err", err)
	}

	if err := uc.broker.PublishToTopic(ctx, "UserLoggedIn", body); err != nil {
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		uc.log.Error("publish confirm email failed", "err", err)
	}

	return access, refresh, nil
}
