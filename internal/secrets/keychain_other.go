//go:build !darwin

package secrets

import "errors"

var ErrUnavailable = errors.New("secret storage unavailable")

func Get(account string) (string, bool, error) {
	return "", false, ErrUnavailable
}

func Set(account, value string) error {
	return ErrUnavailable
}

func Delete(account string) error {
	return ErrUnavailable
}
