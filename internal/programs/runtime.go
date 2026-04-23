package programs

import (
	"strings"
	"time"
)

type LogEntry struct {
	Stream    string `json:"stream"`
	Line      string `json:"line"`
	Timestamp string `json:"timestamp"`
}

type LogQuery struct {
	Limit  int    `json:"limit"`
	Stream string `json:"stream"`
}

type LogView struct {
	ProgramID string     `json:"programId"`
	Entries   []LogEntry `json:"entries"`
	Truncated bool       `json:"truncated"`
	Total     int        `json:"total"`
}

type logBuffer struct {
	limit     int
	truncated bool
	entries   []LogEntry
	total     int
}

func newLogBuffer(limit int) *logBuffer {
	if limit <= 0 {
		limit = 500
	}

	return &logBuffer{limit: limit}
}

func (b *logBuffer) Append(stream string, line string, ts time.Time) {
	entry := LogEntry{
		Stream:    stream,
		Line:      line,
		Timestamp: formatTimestamp(ts),
	}

	b.total++
	if len(b.entries) == b.limit {
		b.entries = append(b.entries[1:], entry)
		b.truncated = true
		return
	}

	b.entries = append(b.entries, entry)
}

func (b *logBuffer) View(query LogQuery) LogView {
	entries := make([]LogEntry, 0, len(b.entries))
	for _, entry := range b.entries {
		if query.Stream != "" && query.Stream != entry.Stream {
			continue
		}
		entries = append(entries, entry)
	}

	if query.Limit > 0 && len(entries) > query.Limit {
		entries = append([]LogEntry(nil), entries[len(entries)-query.Limit:]...)
	} else {
		entries = append([]LogEntry(nil), entries...)
	}

	return LogView{
		Entries:   entries,
		Truncated: b.truncated,
		Total:     b.total,
	}
}

func (b *logBuffer) LastNonEmptyLine(stream string) string {
	fallback := ""

	for index := len(b.entries) - 1; index >= 0; index-- {
		entry := b.entries[index]
		if stream != "" && entry.Stream != stream {
			continue
		}

		line := strings.TrimSpace(entry.Line)
		if line == "" {
			continue
		}
		if fallback == "" {
			fallback = line
		}
		if strings.Contains(strings.ToLower(line), "error:") {
			return line
		}
	}

	return fallback
}
