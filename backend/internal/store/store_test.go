package store

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
)

func openTemp(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sub", "t.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}

func TestNormalize(t *testing.T) {
	tests := []struct{ in, want string }{
		{"ripe", "ripe"},
		{" RIPE\n", "ripe"},
		{"Ripe.", "ripe"},
		{"Overripe!", "overripe"},
		{"unripe", "unripe"},
		{"Not  Fruit", "not fruit"},
		{"not-fruit", "not fruit"},
		{"\"ripe\"", "ripe"},
		{"banana", "other"},
		{"", "other"},
	}
	for _, tt := range tests {
		if got := Normalize(tt.in); got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCountsEmpty(t *testing.T) {
	s, _ := openTemp(t)
	got, err := s.Counts(context.Background())
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if len(got) != len(Labels) {
		t.Errorf("len(counts) = %d, want %d", len(got), len(Labels))
	}
	for _, l := range Labels {
		n, ok := got[l]
		if !ok || n != 0 {
			t.Errorf("counts[%q] = %d (present=%v), want 0", l, n, ok)
		}
	}
}

func TestRecordAndCount(t *testing.T) {
	s, _ := openTemp(t)
	ctx := context.Background()
	for _, r := range []struct{ source, raw string }{
		{"mqtt", "Ripe."},
		{"http", "ripe"},
		{"mqtt", "unripe"},
		{"http", "???"},
	} {
		if err := s.Record(ctx, r.source, r.raw); err != nil {
			t.Fatalf("Record(%q, %q): %v", r.source, r.raw, err)
		}
	}
	got, err := s.Counts(ctx)
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	want := map[string]int{"ripe": 2, "unripe": 1, "overripe": 0, "not fruit": 0, "other": 1}
	for l, w := range want {
		if got[l] != w {
			t.Errorf("counts[%q] = %d, want %d", l, got[l], w)
		}
	}
}

func TestCountsPersistAcrossReopen(t *testing.T) {
	s, path := openTemp(t)
	ctx := context.Background()
	if err := s.Record(ctx, "mqtt", "ripe"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	got, err := s2.Counts(ctx)
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if got["ripe"] != 1 {
		t.Errorf("ripe = %d after reopen, want 1", got["ripe"])
	}
}

func TestRecordConcurrent(t *testing.T) {
	s, _ := openTemp(t)
	ctx := context.Background()
	const n = 50
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- s.Record(ctx, "mqtt", "ripe")
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("Record: %v", err)
		}
	}
	got, err := s.Counts(ctx)
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if got["ripe"] != n {
		t.Errorf("ripe = %d, want %d", got["ripe"], n)
	}
}
