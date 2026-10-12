package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnalyzeSendsKeyInHeaderAndThinkingLevel(t *testing.T) {
	const key = "secret-key"
	var gotPath, gotQuery, gotKey string
	var gotBody generateContentRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery, gotKey = r.URL.Path, r.URL.RawQuery, r.Header.Get("x-goog-api-key")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":" ripe\n"}]}}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{APIKey: key, Model: "m", Endpoint: srv.URL, ThinkingLevel: "low"})
	got, err := c.Analyze(context.Background(), []byte("img"), "image/png")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if got != "ripe" {
		t.Errorf("result = %q, want %q", got, "ripe")
	}
	if gotPath != "/m:generateContent" {
		t.Errorf("path = %q", gotPath)
	}
	if gotKey != key {
		t.Errorf("x-goog-api-key = %q, want %q", gotKey, key)
	}
	if strings.Contains(gotQuery, key) {
		t.Errorf("API key leaked into URL query %q", gotQuery)
	}
	if gotBody.GenerationConfig == nil || gotBody.GenerationConfig.ThinkingConfig == nil ||
		gotBody.GenerationConfig.ThinkingConfig.ThinkingLevel != "low" {
		t.Errorf("thinkingLevel not sent: %+v", gotBody.GenerationConfig)
	}
}

func TestAnalyzeOmitsGenerationConfigWithoutThinkingLevel(t *testing.T) {
	var raw map[string]json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&raw)
		w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ripe"}]}}]}`))
	}))
	defer srv.Close()

	c := NewClient(Config{APIKey: "k", Endpoint: srv.URL})
	if _, err := c.Analyze(context.Background(), []byte("img"), "image/png"); err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if _, ok := raw["generationConfig"]; ok {
		t.Errorf("generationConfig sent without a thinking level: %s", raw["generationConfig"])
	}
}
