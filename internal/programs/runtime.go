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
	lastRelevant map[string]string
}

func newLogBuffer(limit int) *logBuffer {
	if limit <= 0 {
		limit = 500
	}

	return &logBuffer{
		limit:        limit,
		lastRelevant: make(map[string]string, 2),
	}
}

func (b *logBuffer) Append(stream string, line string, ts time.Time) {
	entry := LogEntry{
		Stream:    stream,
		Line:      line,
		Timestamp: formatTimestamp(ts),
	}

	trimmed := strings.TrimSpace(line)
	if trimmed != "" {
		if scoreLogLine(trimmed) >= 0 {
			b.lastRelevant[stream] = trimmed
		}
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
	if stream != "" {
		if line := strings.TrimSpace(b.lastRelevant[stream]); line != "" {
			return line
		}
	}

	fallback := ""
	bestScore := -1
	bestLine := ""

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

		score := scoreLogLine(line)
		if score > bestScore {
			bestScore = score
			bestLine = line
		}
	}

	if bestLine != "" {
		return bestLine
	}
	return fallback
}

func scoreLogLine(line string) int {
	lower := strings.ToLower(strings.TrimSpace(line))
	if lower == "" {
		return -1
	}

	score := 0
	if strings.Contains(lower, "error:") {
		score += 100
	}
	if strings.Contains(lower, "cannot find") || strings.Contains(lower, "failed") || strings.Contains(lower, "exception") {
		score += 80
	}
	if strings.Contains(lower, "module") || strings.Contains(lower, "denied") || strings.Contains(lower, "not found") {
		score += 25
	}
	if strings.HasPrefix(lower, "at ") || strings.Contains(lower, "/loader.js:") || strings.Contains(lower, ".js:") {
		score -= 40
	}
	if strings.Contains(lower, "exit status") {
		score -= 20
	}

	return score
}
