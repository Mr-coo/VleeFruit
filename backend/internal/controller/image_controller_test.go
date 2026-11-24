package controller

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/gin-gonic/gin"
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

// postImage uploads a tiny PNG-typed part to a fresh router and returns the status.
func postImage(t *testing.T, ic *ImageController) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/images", ic.Upload)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="image"; filename="a.png"`)
	h.Set("Content-Type", "image/png")
	part, err := mw.CreatePart(h)
	if err != nil {
		t.Fatalf("CreatePart: %v", err)
	}
	if _, err := part.Write([]byte("fakepng")); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/images", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestUploadRecordsResult(t *testing.T) {
	rec := &fakeRecorder{}
	ic := NewImageController(fakeAnalyzer{result: "Overripe"}, rec)
	if code := postImage(t, ic); code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if rec.calls != 1 || rec.source != "http" || rec.raw != "Overripe" {
		t.Errorf("recorder = %+v, want one call (http, Overripe)", *rec)
	}
}

func TestUploadAnalyzerErrorNotRecorded(t *testing.T) {
	rec := &fakeRecorder{}
	ic := NewImageController(fakeAnalyzer{err: errors.New("boom")}, rec)
	if code := postImage(t, ic); code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", code)
	}
	if rec.calls != 0 {
		t.Errorf("recorder calls = %d, want 0", rec.calls)
	}
}

func TestUploadRecorderErrorStill200(t *testing.T) {
	rec := &fakeRecorder{err: errors.New("db down")}
	ic := NewImageController(fakeAnalyzer{result: "Ripe"}, rec)
	if code := postImage(t, ic); code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
}

func TestUploadNilRecorder(t *testing.T) {
	ic := NewImageController(fakeAnalyzer{result: "Ripe"}, nil)
	if code := postImage(t, ic); code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
}
