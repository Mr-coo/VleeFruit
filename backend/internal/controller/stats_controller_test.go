package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeCounter struct {
	counts map[string]int
	err    error
}

func (f fakeCounter) Counts(ctx context.Context) (map[string]int, error) {
	return f.counts, f.err
}

// getStats issues GET /stats against a fresh router and returns the recorder.
func getStats(t *testing.T, sc *StatsController) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/stats", sc.Stats)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stats", nil))
	return w
}

func TestStatsOK(t *testing.T) {
	want := map[string]int{"ripe": 2, "unripe": 1, "overripe": 0, "not fruit": 3, "other": 0}
	w := getStats(t, NewStatsController(fakeCounter{counts: want}))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body struct {
		Counts map[string]int `json:"counts"`
		Total  int            `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !reflect.DeepEqual(body.Counts, want) {
		t.Errorf("counts = %v, want %v", body.Counts, want)
	}
	if body.Total != 6 {
		t.Errorf("total = %d, want 6", body.Total)
	}
}

func TestStatsCounterError(t *testing.T) {
	w := getStats(t, NewStatsController(fakeCounter{err: errors.New("boom")}))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] == "" {
		t.Errorf("expected error message, got %v", body)
	}
}

func TestStatsNilCounter(t *testing.T) {
	w := getStats(t, NewStatsController(nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "stats storage unavailable" {
		t.Errorf("error = %q", body["error"])
	}
}
