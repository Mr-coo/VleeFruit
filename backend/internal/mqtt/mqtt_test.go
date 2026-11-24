package mqtt

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeAnalyzer struct {
	result string
	err    error
}

func (f fakeAnalyzer) Analyze(ctx context.Context, imageData []byte, mimeType string) (string, error) {
	return f.result, f.err
}

type fakeRecorder struct {
	calls  int
	source string
	raw    string
	err    error
}

func (f *fakeRecorder) Record(ctx context.Context, source, raw string) error {
	f.calls++
	f.source = source
	f.raw = raw
	return f.err
}

func TestRecordImageResult(t *testing.T) {
	ctx := context.Background()
	img := []byte("img")

	t.Run("records success", func(t *testing.T) {
		rec := &fakeRecorder{}
		c := NewClient(Config{}, fakeAnalyzer{result: "ripe"}, rec)
		if got := c.analyzeAndRecord(ctx, img); got != "ripe" {
			t.Errorf("payload = %q, want %q", got, "ripe")
		}
		if rec.calls != 1 || rec.source != "mqtt" || rec.raw != "ripe" {
			t.Errorf("recorder = %+v, want one call (mqtt, ripe)", *rec)
		}
	})

	t.Run("analyzer error is not recorded", func(t *testing.T) {
		rec := &fakeRecorder{}
		c := NewClient(Config{}, fakeAnalyzer{err: errors.New("boom")}, rec)
		got := c.analyzeAndRecord(ctx, img)
		if !strings.HasPrefix(got, "error: ") {
			t.Errorf("payload = %q, want error prefix", got)
		}
		if rec.calls != 0 {
			t.Errorf("recorder calls = %d, want 0", rec.calls)
		}
	})

	t.Run("recorder error still returns result", func(t *testing.T) {
		rec := &fakeRecorder{err: errors.New("db down")}
		c := NewClient(Config{}, fakeAnalyzer{result: "ripe"}, rec)
		if got := c.analyzeAndRecord(ctx, img); got != "ripe" {
			t.Errorf("payload = %q, want %q", got, "ripe")
		}
	})

	t.Run("nil recorder", func(t *testing.T) {
		c := NewClient(Config{}, fakeAnalyzer{result: "ripe"}, nil)
		if got := c.analyzeAndRecord(ctx, img); got != "ripe" {
			t.Errorf("payload = %q, want %q", got, "ripe")
		}
	})
}
