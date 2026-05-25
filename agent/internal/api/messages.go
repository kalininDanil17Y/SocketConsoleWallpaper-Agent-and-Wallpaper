package api

import "socket-console-agent/internal/metrics"

type helloMessage struct {
	Type    string `json:"type"`
	Version string `json:"version"`
	Name    string `json:"name"`
}

type metricsMessage struct {
	Type   string         `json:"type"`
	Status metrics.Status `json:"status"`
}

type configMessage struct {
	Type   string      `json:"type"`
	Config interface{} `json:"config"`
}

type errorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
