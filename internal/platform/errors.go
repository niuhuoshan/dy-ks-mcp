package platform

import (
	"errors"
	"fmt"
	"strings"
)

var ErrNotImplemented = errors.New("not implemented")

type NotImplementedError struct {
	Platform string
	Method   string
}

func (e *NotImplementedError) Error() string {
	platform := strings.TrimSpace(e.Platform)
	method := strings.TrimSpace(e.Method)

	switch {
	case platform != "" && method != "":
		return fmt.Sprintf("%s.%s: %s", platform, method, ErrNotImplemented)
	case platform != "":
		return fmt.Sprintf("%s: %s", platform, ErrNotImplemented)
	case method != "":
		return fmt.Sprintf("%s: %s", method, ErrNotImplemented)
	default:
		return ErrNotImplemented.Error()
	}
}

func (e *NotImplementedError) Unwrap() error {
	return ErrNotImplemented
}

func NewNotImplementedError(platform string, method string) error {
	return &NotImplementedError{
		Platform: strings.ToLower(strings.TrimSpace(platform)),
		Method:   strings.TrimSpace(method),
	}
}

func IsNotImplemented(err error) bool {
	return errors.Is(err, ErrNotImplemented)
}
