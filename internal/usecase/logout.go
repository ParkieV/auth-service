package usecase

import (
	"context"
	"encoding/json"
	"github.com/ParkieV/auth-service/internal/infrastructure/auth_client"
	"github.com/ParkieV/auth-service/internal/infrastructure/broker"
	"github.com/ParkieV/auth-service/internal/infrastructure/cache"
	"log/slog"
)

type LogoutUsecase struct {
	ac     auth_client.AuthClient
	broker broker.MessageBroker
	cache  cache.Cache
	log    *slog.Logger
}

func NewLogoutUsecase(ac auth_client.AuthClient, broker broker.MessageBroker, cache cache.Cache, log *slog.Logger) *LogoutUsecase {
	return &LogoutUsecase{ac: ac, broker: broker, cache: cache, log: log}
}

func (uc *LogoutUsecase) Logout(ctx context.Context, userID, refresh string) error {
	if err := uc.ac.Logout(ctx, refresh); err != nil {
		uc.log.Error("logout failed", "err", err)
		return err
	}

	err := uc.cache.Delete(ctx, refresh)
	if err != nil {
		uc.log.WarnContext(ctx, "cache remove failed", "err", err)
	}

	msg := struct {
		UserID string `json:"user_id"`
	}{
		UserID: userID,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		uc.log.Error("marshal confirm payload failed", "err", err)
	}

	if err := uc.broker.PublishToTopic(ctx, "UserLoggedOut", body); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		uc.log.Error("publish confirm email failed", "err", err)
	}

	return nil
}
