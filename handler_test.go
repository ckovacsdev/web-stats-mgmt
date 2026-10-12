package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ckovacs/web-stats-mgmt/internal/pagespeed"
)

func post(t *testing.T, psi *pagespeed.Client, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/scores", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := newApp(psi).Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("response is not JSON: %s", raw)
	}
	return resp.StatusCode, out
}

func TestScoresValidation(t *testing.T) {
	cases := map[string]string{
		"missing token":  `{"url":"https://example.com"}`,
		"missing url":    `{"token":"abc"}`,
		"non-http url":   `{"url":"ftp://example.com","token":"abc"}`,
		"relative url":   `{"url":"example.com","token":"abc"}`,
		"bad strategy":   `{"url":"https://example.com","token":"abc","strategy":"tablet"}`,
		"malformed json": `{"url":`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			code, out := post(t, pagespeed.New(), body)
			if code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", code)
			}
			if out["error"] == "" || out["error"] == nil {
				t.Error("expected error message")
			}
		})
	}
}

func TestScoresSuccessAndUpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Goog-Api-Key") != "good" {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"message":"Quota exceeded"}}`))
			return
		}
		if r.URL.Query().Get("strategy") != "DESKTOP" {
			t.Errorf("strategy = %q, want DESKTOP", r.URL.Query().Get("strategy"))
		}
		w.Write([]byte(`{"lighthouseResult":{"requestedUrl":"https://example.com","categories":{"performance":{"score":0.5}}}}`))
	}))
	defer upstream.Close()
	psi := &pagespeed.Client{HTTP: upstream.Client(), Endpoint: upstream.URL}

	code, out := post(t, psi, `{"url":"https://example.com","token":"good","strategy":"desktop"}`)
	if code != http.StatusOK {
		t.Fatalf("status = %d, body = %v", code, out)
	}
	scores := out["scores"].(map[string]any)
	if scores["performance"] != float64(50) || scores["seo"] != nil {
		t.Errorf("unexpected scores: %v", scores)
	}

	code, out = post(t, psi, `{"url":"https://example.com","token":"bad"}`)
	if code != http.StatusTooManyRequests || out["error"] != "Quota exceeded" {
		t.Errorf("status = %d, body = %v", code, out)
	}
}
