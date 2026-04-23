package programs

import (
	"testing"
	"time"
)

func TestLogBufferKeepsMostRecentEntries(t *testing.T) {
	buffer := newLogBuffer(3)
	buffer.Append("stdout", "one", time.Unix(1, 0))
	buffer.Append("stdout", "two", time.Unix(2, 0))
	buffer.Append("stderr", "three", time.Unix(3, 0))
	buffer.Append("stdout", "four", time.Unix(4, 0))

	view := buffer.View(LogQuery{Limit: 10})
	if len(view.Entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(view.Entries))
	}
	if view.Entries[0].Line != "two" || view.Entries[2].Line != "four" {
		t.Fatalf("entries = %#v", view.Entries)
	}
	if !view.Truncated {
		t.Fatal("Truncated = false, want true")
	}
}
