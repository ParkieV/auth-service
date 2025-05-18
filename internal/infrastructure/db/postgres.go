package db

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/stdlib"

	"github.com/ParkieV/auth-service/internal/config"
	"github.com/ParkieV/auth-service/internal/domain"
)

// Postgres хранит подключение и реализует методы UserRepository
type Postgres struct {
	db *sql.DB
}

// NewPostgres открывает соединение по параметрам из cfg и проверяет его.
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

// DB возвращает *sql.DB для тестов или миграций
func (p *Postgres) DB() *sql.DB {
	return p.db
}

// Save сохраняет пользователя в таблицу users.
func (p *Postgres) Save(u *domain.User) error {
	const q = `
INSERT INTO users
  (id, email, password_hash, confirmation_id, expires_at, confirmed)
VALUES ($1,$2,$3,$4,$5,$6)
`
	_, err := p.db.Exec(
		q,
		u.ID,
		u.Email.String(),
		u.PasswordHash,
		u.ConfirmationID,
		u.ExpiresAt,
		u.Confirmed,
	)
	return err
}

// FindByEmail ищет пользователя по email
func (p *Postgres) FindByEmail(email domain.Email) (*domain.User, error) {
	const q = `
SELECT id,email,password_hash,confirmation_id,expires_at,confirmed
  FROM users
 WHERE email = $1
`
	row := p.db.QueryRow(q, email.String())
	var (
		id, emailStr, pwdHash, code string
		expiresAt                   = new(sql.NullTime)
		confirmed                   = new(bool)
	)
	if err := row.Scan(&id, &emailStr, &pwdHash, &code, expiresAt, confirmed); err != nil {
		return nil, err
	}

	em, err := domain.NewEmail(emailStr)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		ID:             id,
		Email:          em,
		PasswordHash:   pwdHash,
		ConfirmationID: code,
		ExpiresAt:      expiresAt.Time,
		Confirmed:      *confirmed,
	}
	return user, nil
}
