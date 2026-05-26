package main

import (
	"fmt"
	"time"

	"github.com/kardianos/service"
)

func serviceState(svc service.Service) (service.Status, bool, error) {
	type result struct {
		status service.Status
		err    error
	}

	ch := make(chan result, 1)
	go func() {
		status, err := svc.Status()
		ch <- result{status: status, err: err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			return service.StatusUnknown, false, res.err
		}
		return res.status, true, nil
	case <-time.After(2 * time.Second):
		return service.StatusUnknown, false, fmt.Errorf("service status check timed out")
	}
}
