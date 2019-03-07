package pkg

import (
	"bytes"
	"testing"
)

// TODO(tvi): Refactor to table driven tests.
func TestParseError(t *testing.T) {
	b := bytes.NewBufferString(`
# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes 7.859624e+06`)
	_, err := parse(b, "name", "key")
	if err == nil {
		t.Fatalf("Expected parse error, got nil")
	}
}

func TestParseSuccess(t *testing.T) {
	b := bytes.NewBufferString(`
# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes 7.859624e+06
`)
	_, err := parse(b, "name", "key")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
