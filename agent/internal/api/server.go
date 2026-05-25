package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"socket-console-agent/internal/asciiart"
	"socket-console-agent/internal/config"
	"socket-console-agent/internal/metrics"
)

type Server struct {
	mu          sync.RWMutex
	cfg         *config.Config
	cfgPath     string
	http        *http.Server
	collector   *metrics.Collector
	ascii       *asciiart.Manager
	hub         *hub
	upgrader    websocket.Upgrader
	lastMetrics []byte
}

func NewServer(cfg *config.Config, cfgPath string) *Server {
	cfg.Normalize()
	return &Server{
		cfg:       cfg,
		cfgPath:   cfgPath,
		collector: metrics.NewCollector(),
		ascii:     asciiart.NewManager(cfg.Images),
		hub:       newHub(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/config", s.handleConfig)
	mux.HandleFunc("/api/v1/interfaces", s.handleInterfaces)
	mux.HandleFunc("/api/v1/disks", s.handleDisks)
	mux.HandleFunc("/api/v1/images", s.handleImages)
	mux.HandleFunc("/api/v1/ascii", s.handleASCII)
	mux.HandleFunc("/api/v1/live", s.handleLive)

	cfg := s.Config()
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.http = &http.Server{
		Addr:              addr,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go s.runMetrics(runCtx)
	go s.ascii.Run(runCtx, func(frame *asciiart.Frame) {
		s.hub.broadcastJSON(frame)
	})
	go func() {
		<-ctx.Done()
		_ = s.Shutdown(context.Background())
	}()

	err = s.http.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.http == nil {
		return nil
	}
	return s.http.Shutdown(ctx)
}

func (s *Server) Config() *config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copied := *s.cfg
	copied.Disks.Include = append([]string(nil), s.cfg.Disks.Include...)
	return &copied
}

func (s *Server) setConfig(cfg *config.Config) {
	cfg.Normalize()
	s.mu.Lock()
	s.cfg = cfg
	s.lastMetrics = nil
	s.mu.Unlock()
	s.ascii.UpdateConfig(cfg.Images)
}

func (s *Server) collectStatus() (metrics.Status, error) {
	return s.collector.Collect(s.Config())
}

func (s *Server) runMetrics(ctx context.Context) {
	s.emitMetricsIfChanged()

	interval := s.metricsInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nextInterval := s.metricsInterval()
			if nextInterval != interval {
				ticker.Reset(nextInterval)
				interval = nextInterval
			}
			s.emitMetricsIfChanged()
		}
	}
}

func (s *Server) emitMetricsIfChanged() {
	status, err := s.collectStatus()
	if err != nil {
		s.hub.broadcastJSON(errorMessage{Type: "error", Message: err.Error()})
		return
	}
	msg := metricsMessage{Type: "metrics", Status: status}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	s.mu.Lock()
	if string(data) == string(s.lastMetrics) {
		s.mu.Unlock()
		return
	}
	s.lastMetrics = append(s.lastMetrics[:0], data...)
	s.mu.Unlock()

	s.hub.broadcast(data)
}

func (s *Server) metricsInterval() time.Duration {
	cfg := s.Config()
	ms := cfg.Metrics.IntervalMs
	if ms < 1000 {
		ms = 5000
	}
	return time.Duration(ms) * time.Millisecond
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	status, err := s.collectStatus()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.Config())
	case http.MethodPost:
		defer r.Body.Close()
		var cfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		cfg.Normalize()
		if err := config.Save(s.cfgPath, &cfg); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		s.setConfig(&cfg)
		s.hub.broadcastJSON(configMessage{Type: "config", Config: cfg})
		writeJSON(w, http.StatusOK, cfg)
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	interfaces, err := metrics.Interfaces()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, interfaces)
}

func (s *Server) handleDisks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	disks, err := metrics.Disks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, disks)
}

func (s *Server) handleImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	images, err := s.ascii.Images()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, images)
}

func (s *Server) handleASCII(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	frame, err := s.ascii.CurrentOrNext()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if frame == nil {
		writeJSON(w, http.StatusNotFound, errorMessage{Type: "error", Message: "no supported images found"})
		return
	}
	writeJSON(w, http.StatusOK, frame)
}

func (s *Server) handleLive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	if err := conn.WriteJSON(helloMessage{Type: "hello", Version: metrics.AgentVersion, Name: metrics.AgentName}); err != nil {
		return
	}
	if status, err := s.collectStatus(); err == nil {
		if err := conn.WriteJSON(metricsMessage{Type: "metrics", Status: status}); err != nil {
			return
		}
	}
	if frame := s.ascii.Current(); frame != nil {
		if err := conn.WriteJSON(frame); err != nil {
			return
		}
	}

	ch := make(chan []byte, 16)
	s.hub.register(ch)
	defer s.hub.unregister(ch)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.NextReader(); err != nil {
				return
			}
		}
	}()

	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-done:
			return
		case data := <-ch:
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ping.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "content-type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, errorMessage{Type: "error", Message: err.Error()})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, errorMessage{Type: "error", Message: "method not allowed"})
}
