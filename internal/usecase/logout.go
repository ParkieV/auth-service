package usecase

import (
	"context"
	"github.com/ParkieV/auth-service/internal/infrastructure/auth_client"
	"github.com/ParkieV/auth-service/internal/infrastructure/cache"
	"log/slog"
)

type LogoutUsecase struct {
	ac    auth_client.AuthClient
	cache cache.Cache
	log   *slog.Logger
}

func NewLogoutUsecase(ac auth_client.AuthClient, cache cache.Cache, log *slog.Logger) *LogoutUsecase {
	return &LogoutUsecase{ac: ac, cache: cache, log: log}
}

func (uc *LogoutUsecase) Logout(ctx context.Context, refresh string) error {
	if err := uc.ac.Logout(ctx, refresh); err != nil {
		uc.log.Error("logout failed", "err", err)
		return err
	}

	err := uc.cache.Delete(ctx, refresh)
	if err != nil {
		uc.log.WarnContext(ctx, "cache remove failed", "err", err)
	}
	return nil
}
