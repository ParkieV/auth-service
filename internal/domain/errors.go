package domain

import "errors"

var (
	// ErrInvalidEmail возвращается, если передан некорректный формат e-mail
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrInvalidConfirmationCode — код подтверждения не совпал
	ErrInvalidConfirmationCode = errors.New("invalid confirmation code")

	// ErrConfirmationExpired — время жизни кода подтверждения истекло
	ErrConfirmationExpired = errors.New("confirmation code expired")

	// ErrAlreadyConfirmed — пользователь уже подтверждён
	ErrAlreadyConfirmed = errors.New("user already confirmed")
)
