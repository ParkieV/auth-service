package domain

import "time"

type User struct {
	ID             string
	Email          Email
	PasswordHash   string
	ConfirmationID string
	ExpiresAt      time.Time
	Confirmed      bool
}

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
