package usecase

import (
	"context"
	"errors"
	"github.com/ParkieV/auth-service/internal/infrastructure/auth_client"
	_ "github.com/ParkieV/auth-service/internal/infrastructure/auth_client"
	"github.com/ParkieV/auth-service/internal/infrastructure/cache"
	"github.com/google/uuid"
	"log/slog"
	"time"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid usecase token")
	ErrRefreshFailed       = errors.New("usecase failed")
)

type RefreshUsecase struct {
	ac         auth_client.AuthClient
	cache      cache.Cache
	refreshTTL time.Duration
	log        *slog.Logger
}

func NewRefreshUsecase(ac auth_client.AuthClient, cache cache.Cache, refreshTTL time.Duration, log *slog.Logger) *RefreshUsecase {
	return &RefreshUsecase{ac: ac, cache: cache, refreshTTL: refreshTTL, log: log}
}

func (uc *RefreshUsecase) Refresh(ctx context.Context, oldRT string) (string, string, error) {

	userID, err := uc.cache.Get(ctx, oldRT)
	if err != nil {
		switch {
		case ctx.Err() != nil:
			return "", "", ctx.Err()
		case errors.Is(err, cache.ErrKeyNotFound):
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
		return "", "", ErrInvalidRefreshToken
	}

	newAT, err := uc.ac.IssueAccessToken(ctx, userID)
	if err != nil {
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		uc.log.Error("issue access failed", "err", err)
		return "", "", ErrRefreshFailed
	}

	return newAT, newRT, nil
}
