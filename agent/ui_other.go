//go:build !windows

package main

import (
	"errors"

	"github.com/kardianos/service"
)

func runUI(svc service.Service) error {
	return errors.New("native UI is available only on Windows")
}
