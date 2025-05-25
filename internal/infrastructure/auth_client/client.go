package auth_client

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/ParkieV/auth-service/internal/config"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthClient interface {
	GenerateTokens(ctx context.Context, userID string) (string, string, error)
	IssueAccessToken(ctx context.Context, userID string) (string, error)
	VerifyAccess(ctx context.Context, accessToken string) (bool, string, error)
	Logout(ctx context.Context, refreshToken string) error
}

type TokenRepository struct {
	db         *sql.DB
	hmacKey    []byte
	ttl        time.Duration
	refreshTTL time.Duration
	log        *slog.Logger
}

func NewDBTokenRepository(pgCfg config.PostgresConfig, jwtCfg config.JWTConfig, log *slog.Logger) (*TokenRepository, error) { // db *sql.DB, hmacKey []byte, ttl, refreshTTL time.Duration, log *slog.Logger) *TokenRepository {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		pgCfg.Host, pgCfg.Port, pgCfg.User, pgCfg.Password, pgCfg.DBName, pgCfg.SSLMode,
	)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		log.Error("Cannot connect to database", "error", err)
		return nil, err
	}
	return &TokenRepository{
		db:         db,
		hmacKey:    jwtCfg.HmacKey(),
		ttl:        jwtCfg.AccessTTL,
		refreshTTL: jwtCfg.RefreshTTL,
		log:        log,
	}, nil
}

func (c *TokenRepository) GenerateTokens(ctx context.Context, userID string) (string, string, error) {
	access, err := c.signJWT(userID)
	if err != nil {
		return "", "", err
	}
	refresh := uuid.NewString()

	const q = `
        INSERT INTO tokens (id, user_id, access_token, refresh_token, expires_at)
        VALUES ($1,$2,$3,$4,$5)`
	_, err = c.db.ExecContext(ctx, q,
		uuid.New(), userID, access, refresh, time.Now().Add(c.refreshTTL))
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (c *TokenRepository) IssueAccessToken(ctx context.Context, userID string) (string, error) {
	access, err := c.signJWT(userID)
	if err != nil {
		return "", err
	}
	const q = `UPDATE tokens SET access_token = $1, expires_at = $2
               WHERE user_id = $3 AND revoked_at IS NULL`
	_, _ = c.db.ExecContext(ctx, q, access, time.Now().Add(c.refreshTTL), userID)
	return access, nil
}

func (c *TokenRepository) VerifyAccess(ctx context.Context, access string) (bool, string, error) {
	claims, err := c.parseJWT(access)
	if err != nil {
		return false, "", err
	}
	var revokedAt sql.NullTime
	err = c.db.QueryRowContext(ctx,
		`SELECT revoked_at FROM tokens WHERE access_token = $1`, access).Scan(&revokedAt)
	if err != nil {
		return false, "", err
	}
	if revokedAt.Valid {
		return false, "", nil
	}
	return true, claims.Subject, nil
}

func (c *TokenRepository) Logout(ctx context.Context, refresh string) error {
	_, err := c.db.ExecContext(ctx,
		`UPDATE tokens SET revoked_at = now() WHERE refresh_token = $1`, refresh)
	return err
}

func (c *TokenRepository) signJWT(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(c.ttl)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(c.hmacKey)
}

func (c *TokenRepository) parseJWT(token string) (*jwt.RegisteredClaims, error) {
	t, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{},
		func(_ *jwt.Token) (any, error) { return c.hmacKey, nil })
	if err != nil || !t.Valid {
		return nil, errors.New("invalid token")
	}
	return t.Claims.(*jwt.RegisteredClaims), nil
}
