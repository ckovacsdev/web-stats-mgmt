// Package pagespeed is a minimal client for the Google PageSpeed Insights v5 API
// that returns only the Lighthouse category scores.
package pagespeed

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"
)

const DefaultEndpoint = "https://www.googleapis.com/pagespeedonline/v5/runPagespeed"

// Lighthouse categories requested from the API.
var categories = []string{"PERFORMANCE", "ACCESSIBILITY", "BEST_PRACTICES", "SEO"}

type Client struct {
	HTTP     *http.Client
	Endpoint string
}

func New() *Client {
	return &Client{
		// Lighthouse runs commonly take 20-60s.
		HTTP:     &http.Client{Timeout: 2 * time.Minute},
		Endpoint: DefaultEndpoint,
	}
}

// Scores are 0-100. A nil value means Lighthouse could not compute that category.
type Scores struct {
	Performance   *int `json:"performance"`
	Accessibility *int `json:"accessibility"`
	BestPractices *int `json:"bestPractices"`
	SEO           *int `json:"seo"`
}

type Result struct {
	URL       string `json:"url"`
	Strategy  string `json:"strategy"`
	FetchTime string `json:"fetchTime"`
	Scores    Scores `json:"scores"`
}

// APIError is a non-200 response from Google.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("pagespeed api: %d: %s", e.StatusCode, e.Message)
}

type psiResponse struct {
	LighthouseResult struct {
		RequestedURL string `json:"requestedUrl"`
		FetchTime    string `json:"fetchTime"`
		Categories   map[string]struct {
			Score *float64 `json:"score"`
		} `json:"categories"`
	} `json:"lighthouseResult"`
}

type psiError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Run executes a PageSpeed Insights analysis of target. strategy is "MOBILE" or "DESKTOP".
// The token is sent as a request header rather than a query parameter so it never
// appears in URLs (and therefore never in transport error messages or proxy logs).
func (c *Client) Run(ctx context.Context, token, target, strategy string) (*Result, error) {
	q := url.Values{}
	q.Set("url", target)
	q.Set("strategy", strategy)
	for _, cat := range categories {
		q.Add("category", cat)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("X-Goog-Api-Key", token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling pagespeed api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var e psiError
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error.Message == "" {
			e.Error.Message = http.StatusText(resp.StatusCode)
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Message: e.Error.Message}
	}

	// The full response is several MB (screenshots, audits); decode only what we need.
	var body psiResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decoding pagespeed response: %w", err)
	}

	lr := body.LighthouseResult
	return &Result{
		URL:       lr.RequestedURL,
		Strategy:  strategy,
		FetchTime: lr.FetchTime,
		Scores: Scores{
			Performance:   toPercent(lr.Categories["performance"].Score),
			Accessibility: toPercent(lr.Categories["accessibility"].Score),
			BestPractices: toPercent(lr.Categories["best-practices"].Score),
			SEO:           toPercent(lr.Categories["seo"].Score),
		},
	}, nil
}

func toPercent(score *float64) *int {
	if score == nil {
		return nil
	}
	v := int(math.Round(*score * 100))
	return &v
}
