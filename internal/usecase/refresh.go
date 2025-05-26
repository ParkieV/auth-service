package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ParkieV/auth-service/internal/infrastructure/auth_client"
	_ "github.com/ParkieV/auth-service/internal/infrastructure/auth_client"
	"github.com/ParkieV/auth-service/internal/infrastructure/broker"
	"github.com/ParkieV/auth-service/internal/infrastructure/cache"
	"github.com/google/uuid"
	"log/slog"
	"time"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshFailed       = errors.New("refresh failed")
)

type RefreshUsecase struct {
	ac         auth_client.AuthClient
	broker     broker.MessageBroker
	cache      cache.Cache
	refreshTTL time.Duration
	log        *slog.Logger
}

func NewRefreshUsecase(ac auth_client.AuthClient, broker broker.MessageBroker, cache cache.Cache, refreshTTL time.Duration, log *slog.Logger) *RefreshUsecase {
	return &RefreshUsecase{ac: ac, broker: broker, cache: cache, refreshTTL: refreshTTL, log: log}
}

func (uc *RefreshUsecase) Refresh(ctx context.Context, oldRT string) (string, string, error) {

	userID, err := uc.cache.Get(ctx, oldRT)
	if err != nil {
		switch {
		case ctx.Err() != nil:
			return "", "", ctx.Err()
		case errors.Is(err, cache.ErrKeyNotFound):
			uc.log.Error("Cannot get refresh token from cache")
			return "", "", ErrInvalidRefreshToken
		default:
			uc.log.Error("cache get failed", "err", err)
			return "", "", err
		}
	}

	newRT := uuid.NewString()
	ok, err := uc.cache.SwapRefresh(ctx, userID, oldRT, newRT, uc.refreshTTL)
	if err != nil {
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		uc.log.Error("cache swap failed", "err", err)
		return "", "", ErrRefreshFailed
	}
	if !ok {
		uc.log.Error("Problem is here")
		return "", "", ErrInvalidRefreshToken
	}

	newAT, err := uc.ac.IssueAccessToken(ctx, userID)
	if err != nil {
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		uc.log.Error("issue access token failed", "err", err)
		return "", "", ErrRefreshFailed
	}

	msg := struct {
		UserID       string `json:"user_id"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}{
		UserID:       userID,
		AccessToken:  newAT,
		RefreshToken: newRT,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		uc.log.Error("marshal confirm payload failed", "err", err)
	}

	if err := uc.broker.PublishToTopic(ctx, "UserTokensRefreshed", body); err != nil {
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		uc.log.Error("publish confirm email failed", "err", err)
	}

	return newAT, newRT, nil
}
