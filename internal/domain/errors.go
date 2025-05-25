package domain

import "errors"

var (
	ErrInvalidEmail = errors.New("invalid email format")

	ErrInvalidConfirmationCode = errors.New("invalid confirmation code")

	ErrConfirmationExpired = errors.New("confirmation code expired")

	ErrAlreadyConfirmed = errors.New("user already confirmed")
)
