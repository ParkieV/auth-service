package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ParkieV/auth-service/internal/domain"
	"github.com/ParkieV/auth-service/internal/infrastructure/db"
)

func TestSaveAndFindByEmail(t *testing.T) {
	pg, err := db.NewPostgres(PGConfig)
	require.NoError(t, err)

	// создаём таблицу вручную
	_, err = pg.DB().Exec(`
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  confirmation_id TEXT NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  confirmed BOOLEAN NOT NULL
);
`)
	require.NoError(t, err)

	em, err := domain.NewEmail("intg@test.com")
	require.NoError(t, err)
	user := domain.NewUser("test-id", em, "hashpwd", "code123", time.Now().Add(24*time.Hour))
	user.Confirmed = false

	// сохраняем и читаем
	require.NoError(t, pg.Save(user))

	got, err := pg.FindByEmail(em)
	require.NoError(t, err)

	require.Equal(t, user.ID, got.ID)
	require.Equal(t, user.Email.String(), got.Email.String())
	require.Equal(t, user.PasswordHash, got.PasswordHash)
	require.Equal(t, user.ConfirmationID, got.ConfirmationID)
	require.Equal(t, user.Confirmed, got.Confirmed)
}
