package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// ErrNotConfigured is returned when no API key is available, so the server runs
// but LLM analysis is disabled.
var ErrNotConfigured = errors.New("no LLM configured (set GEMINI_API_KEY)")

// defaultEndpoint is Google's Generative Language REST API base. The model and
// API key are appended per request.
const defaultEndpoint = "https://generativelanguage.googleapis.com/v1beta/models"

// defaultPrompt constrains the vision model to a single ripeness label.
const defaultPrompt = "Classify the fruit in this image by ripeness. " +
	"Reply with exactly one of these words and nothing else: unripe, ripe, overripe, not fruit. " +
	"If the image does not clearly show a fruit, reply: not fruit."

// Config configures the Gemini vision client.
type Config struct {
	// APIKey is the Google Generative Language API key. Empty disables the LLM.
	APIKey string
	// Model is the Gemini model id (e.g. "gemini-2.0-flash").
	Model string
	// Endpoint overrides the API base URL (mainly for tests).
	Endpoint string
	// Prompt is the instruction sent alongside every image.
	Prompt string
	// Timeout bounds a single request.
	Timeout time.Duration
}

// Client talks to the Gemini vision API. A zero-key client is disabled and
// returns ErrNotConfigured from Analyze.
type Client struct {
	cfg  Config
	http *http.Client
}

// NewClient builds a Gemini client, applying sensible defaults. A missing API
// key yields a disabled client so the server still starts.
func NewClient(cfg Config) *Client {
	if cfg.APIKey == "" {
		log.Printf("llm: no GEMINI_API_KEY set; image analysis disabled")
	}
	if cfg.Model == "" {
		cfg.Model = "gemini-2.0-flash"
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = defaultEndpoint
	}
	if cfg.Prompt == "" {
		cfg.Prompt = defaultPrompt
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

// Enabled reports whether an API key is configured.
func (c *Client) Enabled() bool { return c.cfg.APIKey != "" }

// generateContentRequest / response mirror the subset of the Gemini REST schema
// we use: one text part plus one inline image part.
type inlineData struct {
	MIMEType string `json:"mime_type"`
	Data     string `json:"data"`
}

type part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *inlineData `json:"inline_data,omitempty"`
}

type content struct {
	Parts []part `json:"parts"`
}

type generateContentRequest struct {
	Contents []content `json:"contents"`
}

type generateContentResponse struct {
	Candidates []struct {
		Content content `json:"content"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Analyze sends the image to Gemini and returns the model's text description.
func (c *Client) Analyze(ctx context.Context, imageData []byte, mimeType string) (string, error) {
	if !c.Enabled() {
		return "", ErrNotConfigured
	}
	if len(imageData) == 0 {
		return "", errors.New("empty image data")
	}

	reqBody := generateContentRequest{
		Contents: []content{{
			Parts: []part{
				{Text: c.cfg.Prompt},
				{InlineData: &inlineData{
					MIMEType: mimeType,
					Data:     base64.StdEncoding.EncodeToString(imageData),
				}},
			},
		}},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", c.cfg.Endpoint, c.cfg.Model, c.cfg.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("call gemini: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var parsed generateContentResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := parsed.Error.Message
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		return "", fmt.Errorf("gemini error (status %d): %s", resp.StatusCode, msg)
	}

	if parsed.PromptFeedback.BlockReason != "" {
		return "", fmt.Errorf("gemini blocked the request: %s", parsed.PromptFeedback.BlockReason)
	}

	text := extractText(parsed)
	if text == "" {
		return "", errors.New("gemini returned no text")
	}
	return text, nil
}

// extractText joins all text parts of the first candidate.
func extractText(resp generateContentResponse) string {
	if len(resp.Candidates) == 0 {
		return ""
	}
	var b strings.Builder
	for _, p := range resp.Candidates[0].Content.Parts {
		b.WriteString(p.Text)
	}
	return strings.TrimSpace(b.String())
}
