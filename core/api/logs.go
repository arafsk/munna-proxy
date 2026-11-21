package api

import (
	"bytes"
	"sync"
	"time"
)

const maxLogEntries = 1000

// LogEntry is one line of engine output.
type LogEntry struct {
	Seq     uint64    `json:"seq"`
	Time    time.Time `json:"time"`
	Message string    `json:"message"`
}

// logBuffer keeps the most recent log lines. It implements io.Writer so a
// slog text handler can write into it; each Write may contain several lines.
type logBuffer struct {
	mu      sync.Mutex
	entries []LogEntry
	nextSeq uint64
	partial []byte
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.partial = append(b.partial, p...)
	for {
		idx := bytes.IndexByte(b.partial, '\n')
		if idx < 0 {
			break
		}
		b.appendLocked(string(b.partial[:idx]))
		b.partial = b.partial[idx+1:]
	}
	b.partial = append([]byte(nil), b.partial...)
	return len(p), nil
}

func (b *logBuffer) appendLocked(line string) {
	if line == "" {
		return
	}
	b.nextSeq++
	b.entries = append(b.entries, LogEntry{Seq: b.nextSeq, Time: time.Now(), Message: line})
	if len(b.entries) > maxLogEntries {
		b.entries = append([]LogEntry(nil), b.entries[len(b.entries)-maxLogEntries:]...)
	}
}

// since returns entries with a sequence number greater than seq.
func (b *logBuffer) since(seq uint64) []LogEntry {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]LogEntry, 0)
	for _, entry := range b.entries {
		if entry.Seq > seq {
			out = append(out, entry)
		}
	}
	return out
}
