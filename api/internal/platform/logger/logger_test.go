package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONFormatAndLevel(t *testing.T) {
	var buf bytes.Buffer
	log := newWith(&buf, "warn", "json")
	log.Info("hidden")
	log.Warn("shown", "request_id", "abc")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("want 1 line (info filtered out), got %q", buf.String())
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if rec["msg"] != "shown" || rec["request_id"] != "abc" {
		t.Errorf("record = %v", rec)
	}
}
