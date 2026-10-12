package pagespeed

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunParsesScoresAndSendsTokenInHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Goog-Api-Key"); got != "secret" {
			t.Errorf("api key header = %q, want %q", got, "secret")
		}
		if r.URL.Query().Has("key") {
			t.Error("token must not be sent as a query parameter")
		}
		if got := r.URL.Query().Get("url"); got != "https://example.com" {
			t.Errorf("url param = %q", got)
		}
		if got := len(r.URL.Query()["category"]); got != 4 {
			t.Errorf("category params = %d, want 4", got)
		}
		w.Write([]byte(`{"lighthouseResult":{
			"requestedUrl":"https://example.com",
			"fetchTime":"2026-10-12T00:00:00.000Z",
			"categories":{
				"performance":{"score":0.874},
				"accessibility":{"score":1},
				"best-practices":{"score":0.96},
				"seo":{"score":null}
			}}}`))
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), Endpoint: srv.URL}
	res, err := c.Run(context.Background(), "secret", "https://example.com", "MOBILE")
	if err != nil {
		t.Fatal(err)
	}

	check := func(name string, got *int, want int) {
		t.Helper()
		if got == nil || *got != want {
			t.Errorf("%s = %v, want %d", name, got, want)
		}
	}
	check("performance", res.Scores.Performance, 87)
	check("accessibility", res.Scores.Accessibility, 100)
	check("bestPractices", res.Scores.BestPractices, 96)
	if res.Scores.SEO != nil {
		t.Errorf("seo = %d, want nil", *res.Scores.SEO)
	}
	if res.Strategy != "MOBILE" || res.URL != "https://example.com" {
		t.Errorf("unexpected result metadata: %+v", res)
	}
}

func TestRunReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":400,"message":"API key not valid. Please pass a valid API key."}}`))
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), Endpoint: srv.URL}
	_, err := c.Run(context.Background(), "bad", "https://example.com", "MOBILE")

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest || apiErr.Message != "API key not valid. Please pass a valid API key." {
		t.Errorf("unexpected api error: %+v", apiErr)
	}
}
