package db

import (
	"database/sql"
	"fmt"

	"github.com/ParkieV/auth-service/internal/config"
	"github.com/ParkieV/auth-service/internal/domain"
)

// Postgres хранит подключение к БД
type Postgres struct {
	db *sql.DB
}

// NewPostgres открывает соединение и пингует БД
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

// UserRepository возвращает реализацию domain.UserRepository
func (p *Postgres) UserRepository() domain.UserRepository {
	return &userRepo{db: p.db}
}

type userRepo struct {
	db *sql.DB
}

// Save сохраняет нового пользователя
func (r *userRepo) Save(u *domain.User) error {
	const q = `
    INSERT INTO users 
      (id, email, password_hash, confirmation_id, expires_at, confirmed)
    VALUES ($1, $2, $3, $4, $5, $6)
    `
	_, err := r.db.Exec(
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

// FindByEmail находит пользователя по email
func (r *userRepo) FindByEmail(email domain.Email) (*domain.User, error) {
	const q = `
    SELECT id, email, password_hash, confirmation_id, expires_at, confirmed
      FROM users
     WHERE email = $1
    `
	row := r.db.QueryRow(q, email.String())

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
