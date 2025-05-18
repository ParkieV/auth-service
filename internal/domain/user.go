package domain

import "time"

// User — агрегат «пользователь»
type User struct {
	ID             string    // уникальный идентификатор (UUID)
	Email          Email     // value-object
	PasswordHash   string    // хэш пароля
	ConfirmationID string    // код для подтверждения e-mail
	ExpiresAt      time.Time // время истечения действия ConfirmationID
	Confirmed      bool      // признак, что e-mail подтверждён
}

// NewUser создаёт новый экземпляр User.
// Все параметры подготавливаются на уровне usecase (генерация UUID, хэширование, TTL).
func NewUser(
	id string,
	email Email,
	passwordHash string,
	confirmationID string,
	expiresAt time.Time,
) *User {
	return &User{
		ID:             id,
		Email:          email,
		PasswordHash:   passwordHash,
		ConfirmationID: confirmationID,
		ExpiresAt:      expiresAt,
		Confirmed:      false,
	}
}

// Confirm пытается подтвердить пользователя по коду.
// – возвращает ErrAlreadyConfirmed, если уже был подтверждён;
// – ErrInvalidConfirmationCode, если код не совпал;
// – ErrConfirmationExpired, если код устарел.
// now передаётся из usecase, чтобы в тестах можно было управлять «текущим» временем.
func (u *User) Confirm(code string, now time.Time) error {
	if u.Confirmed {
		return ErrAlreadyConfirmed
	}
	if code != u.ConfirmationID {
		return ErrInvalidConfirmationCode
	}
	if now.After(u.ExpiresAt) {
		return ErrConfirmationExpired
	}
	u.Confirmed = true
	return nil
}
