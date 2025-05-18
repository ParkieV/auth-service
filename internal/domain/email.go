package domain

import (
	"fmt"
	"regexp"
)

var emailRegex = regexp.MustCompile(
	`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`,
)

// Email — value object для адреса электронной почты
type Email struct {
	value string
}

// NewEmail валидирует строку и возвращает Email или ErrInvalidEmail
func NewEmail(v string) (Email, error) {
	if !emailRegex.MatchString(v) {
		return Email{}, fmt.Errorf("%w: %s", ErrInvalidEmail, v)
	}
	return Email{value: v}, nil
}

// String возвращает оригинальную строку e-mail
func (e Email) String() string {
	return e.value
}
