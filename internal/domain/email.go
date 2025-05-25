package domain

import (
	"fmt"
	"regexp"
)

var emailRegex = regexp.MustCompile(
	`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`,
)

type Email struct {
	value string
}

func NewEmail(v string) (Email, error) {
	if !emailRegex.MatchString(v) {
		return Email{}, fmt.Errorf("%w: %s", ErrInvalidEmail, v)
	}
	return Email{value: v}, nil
}

func (e Email) String() string {
	return e.value
}
