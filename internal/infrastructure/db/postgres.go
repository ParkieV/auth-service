package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/stdlib"

	"github.com/ParkieV/auth-service/internal/config"
	"github.com/ParkieV/auth-service/internal/domain"
)

var (
	ErrDuplicateKey = errors.New("duplicate key")
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error)
}

type UserMutRepository interface {
	UserRepository
	Save(ctx context.Context, u *domain.User) error
	UpdatePasswordHash(ctx context.Context, userID, newHash string) error
}

type Postgres struct{ db *sql.DB }

func NewPostgres(cfg config.PostgresConfig) (*Postgres, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Postgres{db: db}, nil
}

func (p *Postgres) DB() *sql.DB { return p.db }

func (p *Postgres) Save(ctx context.Context, u *domain.User) error {
	const q = `
	INSERT INTO users
	  (id,  email,  password_hash, confirmation_id, expires_at, confirmed)
	VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := p.db.ExecContext(
		ctx, q,
		u.ID(),
		u.Email().String(),
		u.HashForStorage(),
		u.ConfirmationID(),
		u.ExpiresAt().UTC(),
		u.IsConfirmed(),
	)
	if err != nil && isDuplicateKey(err) {
		return ErrDuplicateKey
	}
	return err
}

func (p *Postgres) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	const q = `
	SELECT id, email, password_hash, confirmation_id, expires_at, confirmed
	  FROM users
	 WHERE email = $1
	`
	row := p.db.QueryRowContext(ctx, q, email.String())

	var (
		id, emailStr, hash, code string
		expires                  time.Time
		confirmed                bool
	)
	if err := row.Scan(&id, &emailStr, &hash, &code, &expires, &confirmed); err != nil {
		return nil, err
	}

	em, err := domain.NewEmail(emailStr)
	if err != nil {
		return nil, err
	}
	user, err := domain.RehydrateUser(id, em, hash, code, expires, confirmed)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (p *Postgres) UpdatePasswordHash(ctx context.Context, userID, newHash string) error {
	const q = `UPDATE users SET password_hash = $1 WHERE id = $2`
	_, err := p.db.ExecContext(ctx, q, newHash, userID)
	return err
}

func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 23505 — unique_violation
		return pgErr.Code == "23505"
	}
	return false
}
