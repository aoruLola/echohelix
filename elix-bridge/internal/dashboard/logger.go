package dashboard

import (
	"sync"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// Logger collects logs in memory
type Logger struct {
	mu      sync.RWMutex
	entries []LogEntry
	maxSize int
}

// NewLogger creates a new logger
func NewLogger(maxSize int) *Logger {
	if maxSize <= 0 {
		maxSize = 500
	}
	return &Logger{
		entries: make([]LogEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Log adds a log entry
func (l *Logger) Log(level, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	l.entries = append(l.entries, entry)

	// 保留最新的 maxSize 条记录
	if len(l.entries) > l.maxSize {
		l.entries = l.entries[len(l.entries)-l.maxSize:]
	}
}

// GetLogs returns the most recent n logs
func (l *Logger) GetLogs(n int) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	total := len(l.entries)
	if n <= 0 || n > total {
		n = total
	}

	// 返回最新的 n 条（倒序）
	result := make([]LogEntry, n)
	for i := 0; i < n; i++ {
		result[i] = l.entries[total-n+i]
	}

	return result
}

// Count returns total log count
func (l *Logger) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries)
}
