package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidEmail            = errors.New("invalid email format")
	ErrInvalidConfirmationCode = errors.New("invalid confirmation code")
	ErrConfirmationExpired     = errors.New("confirmation code expired")
	ErrAlreadyConfirmed        = errors.New("user already confirmed")
)

type User struct {
	id             string
	email          Email
	password       Password
	confirmationID string
	expiresAt      time.Time
	confirmed      bool
}

func (u *User) ID() string             { return u.id }
func (u *User) Email() Email           { return u.email }
func (u *User) IsConfirmed() bool      { return u.confirmed }
func (u *User) ConfirmationID() string { return u.confirmationID }
func (u *User) ExpiresAt() time.Time   { return u.expiresAt }

func NewUserFromRegistration(
	id string,
	email Email,
	plainPwd string,
	confirmationID string,
	ttl time.Duration,
) (*User, error) {

	pwd, err := NewPasswordFromPlain(plainPwd)
	if err != nil {
		return nil, err
	}

	return &User{
		id:             id,
		email:          email,
		password:       pwd,
		confirmationID: confirmationID,
		expiresAt:      time.Now().UTC().Add(ttl),
		confirmed:      false,
	}, nil
}

func RehydrateUser(
	id string,
	email Email,
	hash string,
	confirmationID string,
	expiresAt time.Time,
	confirmed bool,
) (*User, error) {

	pwdVO, err := NewPasswordFromHash(hash)
	if err != nil {
		return nil, err
	}

	return &User{
		id:             id,
		email:          email,
		password:       pwdVO,
		confirmationID: confirmationID,
		expiresAt:      expiresAt.UTC(),
		confirmed:      confirmed,
	}, nil
}

func (u *User) VerifyPassword(plain string) (ok, needRehash bool) {
	if !u.password.Verify(plain) {
		return false, false
	}
	return true, u.password.NeedsRehash()
}

func (u *User) ApplyRehash(newPwd Password) {
	u.password = newPwd
}

func (u *User) Confirm(code string, now time.Time) error {
	switch {
	case u.confirmed:
		return ErrAlreadyConfirmed
	case code != u.confirmationID:
		return ErrInvalidConfirmationCode
	case now.UTC().After(u.expiresAt):
		return ErrConfirmationExpired
	}
	u.confirmed = true
	return nil
}

func (u *User) HashForStorage() string {
	return u.password.Hash()
}
