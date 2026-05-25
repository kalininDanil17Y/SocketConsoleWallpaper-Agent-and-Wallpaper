package asciiart

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"socket-console-agent/internal/config"
)

type Manager struct {
	mu      sync.RWMutex
	cfg     config.ImagesConfig
	cache   map[string]*Frame
	current *Frame
	index   int
}

func NewManager(cfg config.ImagesConfig) *Manager {
	return &Manager{
		cfg:   cfg,
		cache: make(map[string]*Frame),
	}
}

func (m *Manager) UpdateConfig(cfg config.ImagesConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg = cfg
	m.cache = make(map[string]*Frame)
	m.current = nil
	m.index = 0
}

func (m *Manager) Current() *Frame {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *Manager) CurrentOrNext() (*Frame, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current != nil {
		return m.current, nil
	}
	frame, _, err := m.nextLocked()
	return frame, err
}

func (m *Manager) Images() ([]ImageInfo, error) {
	m.mu.RLock()
	dir := m.cfg.Directory
	m.mu.RUnlock()
	return ListImages(dir)
}

func (m *Manager) Run(ctx context.Context, broadcast func(*Frame)) {
	m.emitNext(broadcast)

	interval := m.interval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nextInterval := m.interval()
			if nextInterval != interval {
				ticker.Reset(nextInterval)
				interval = nextInterval
			}
			m.emitNext(broadcast)
		}
	}
}

func (m *Manager) emitNext(broadcast func(*Frame)) {
	frame, changed := m.Next()
	if frame != nil && changed {
		broadcast(frame)
	}
}

func (m *Manager) Next() (*Frame, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	frame, changed, err := m.nextLocked()
	if err != nil || frame == nil {
		return nil, false
	}
	return frame, changed
}

func (m *Manager) nextLocked() (*Frame, bool, error) {
	paths, err := imagePaths(m.cfg.Directory)
	if err != nil || len(paths) == 0 {
		return nil, false, err
	}

	if m.index >= len(paths) {
		m.index = 0
	}
	path := paths[m.index]
	m.index = (m.index + 1) % len(paths)

	stat, err := os.Stat(path)
	if err != nil {
		return nil, false, err
	}
	cacheKey := cacheKey(path, stat.ModTime(), m.cfg)
	frame, ok := m.cache[cacheKey]
	if !ok {
		frame, err = Convert(path, m.cfg)
		if err != nil {
			return nil, false, err
		}
		m.cache[cacheKey] = frame
	}

	if m.current == frame {
		return frame, false, nil
	}
	m.current = frame
	return frame, true, nil
}

func (m *Manager) interval() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	seconds := m.cfg.ChangeEverySeconds
	if seconds <= 0 {
		seconds = 5
	}
	return time.Duration(seconds) * time.Second
}

func ListImages(dir string) ([]ImageInfo, error) {
	paths, err := imagePaths(dir)
	if err != nil {
		return nil, err
	}
	images := make([]ImageInfo, 0, len(paths))
	for _, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			continue
		}
		images = append(images, ImageInfo{
			Name:      filepath.Base(path),
			Path:      path,
			ModTime:   stat.ModTime().Format(time.RFC3339),
			SizeBytes: stat.Size(),
		})
	}
	return images, nil
}

func imagePaths(dir string) ([]string, error) {
	if dir == "" {
		return nil, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if SupportedImage(path) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func cacheKey(path string, modTime time.Time, cfg config.ImagesConfig) string {
	return fmt.Sprintf("%s|%s|%dx%d|%s|%d",
		path,
		modTime.UTC().Format(time.RFC3339Nano),
		cfg.ASCIIWidth,
		cfg.ASCIIHeight,
		cfg.Charset,
		cfg.PaletteSize,
	)
}
