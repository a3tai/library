//go:build darwin

package secrets

import (
	"errors"
	"os/exec"
	"strings"
)

const service = "A3T Library"

var ErrUnavailable = errors.New("secret storage unavailable")

func Get(account string) (string, bool, error) {
	out, err := exec.Command("/usr/bin/security", "find-generic-password", "-a", account, "-s", service, "-w").Output()
	if err != nil {
		return "", false, nil
	}
	value := strings.TrimRight(string(out), "\r\n")
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

func Set(account, value string) error {
	if strings.TrimSpace(value) == "" {
		return Delete(account)
	}
	return exec.Command("/usr/bin/security", "add-generic-password", "-a", account, "-s", service, "-w", value, "-U").Run()
}

func Delete(account string) error {
	err := exec.Command("/usr/bin/security", "delete-generic-password", "-a", account, "-s", service).Run()
	if err != nil {
		return nil
	}
	return nil
}
