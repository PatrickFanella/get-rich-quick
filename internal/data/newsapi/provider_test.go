package newsapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestProviderGetNews(t *testing.T) {
	t.Parallel()

	type requestDetails struct {
		method string
		path   string
		query  url.Values
		apiKey string
		accept string
	}

	from := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, time.January, 2, 23, 59, 59, 0, time.UTC)

	requests := make(chan requestDetails, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- requestDetails{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.Query(),
			apiKey: r.Header.Get("X-Api-Key"),
			accept: r.Header.Get("Accept"),
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"ok",
			"totalResults":2,
			"articles":[
				{
					"source":{"name":"Reuters"},
					"title":"Apple jumps on earnings",
					"description":"Apple reported strong quarterly revenue.",
					"url":"https://example.com/aapl-1",
					"publishedAt":"2024-01-01T14:30:00Z"
				},
				{
					"source":{"name":"Bloomberg"},
					"title":"Apple supply chain update",
					"description":"Suppliers expect steady demand.",
					"url":"https://example.com/aapl-2",
					"publishedAt":"2024-01-02T15:45:00Z"
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient("test-key", discardLogger())
	client.baseURL = server.URL
	client.httpClient = server.Client()
	provider := NewProvider(client)

	got, err := provider.GetNews(context.Background(), "AAPL", from, to)
	if err != nil {
		t.Fatalf("GetNews() error = %v", err)
	}

	want := []data.NewsArticle{
		{
			Title:       "Apple jumps on earnings",
			Summary:     "Apple reported strong quarterly revenue.",
			URL:         "https://example.com/aapl-1",
			Source:      "Reuters",
			PublishedAt: time.Date(2024, time.January, 1, 14, 30, 0, 0, time.UTC),
		},
		{
			Title:       "Apple supply chain update",
			Summary:     "Suppliers expect steady demand.",
			URL:         "https://example.com/aapl-2",
			Source:      "Bloomberg",
			PublishedAt: time.Date(2024, time.January, 2, 15, 45, 0, 0, time.UTC),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetNews() = %#v, want %#v", got, want)
	}

	select {
	case request := <-requests:
		if request.method != http.MethodGet {
			t.Fatalf("request method = %s, want %s", request.method, http.MethodGet)
		}
		if request.path != everythingEndpoint {
			t.Fatalf("request path = %s, want %s", request.path, everythingEndpoint)
		}
		if request.query.Get("q") != "AAPL" {
			t.Fatalf("q = %q, want %q", request.query.Get("q"), "AAPL")
		}
		if request.query.Get("from") != from.Format(time.RFC3339) {
			t.Fatalf("from = %q, want %q", request.query.Get("from"), from.Format(time.RFC3339))
		}
		if request.query.Get("to") != to.Format(time.RFC3339) {
			t.Fatalf("to = %q, want %q", request.query.Get("to"), to.Format(time.RFC3339))
		}
		if request.query.Get("pageSize") != "100" {
			t.Fatalf("pageSize = %q, want %q", request.query.Get("pageSize"), "100")
		}
		if request.apiKey != "test-key" {
			t.Fatalf("X-Api-Key = %q, want %q", request.apiKey, "test-key")
		}
		if request.accept != "application/json" {
			t.Fatalf("Accept = %q, want %q", request.accept, "application/json")
		}
	case <-time.After(time.Second):
		t.Fatal("request details were not captured")
	}
}

func TestProviderGetNewsReturnsErrorForNilClient(t *testing.T) {
	t.Parallel()

	provider := NewProvider(nil)

	_, err := provider.GetNews(
		context.Background(),
		"AAPL",
		time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("GetNews() error = nil, want non-nil")
	}
	if err.Error() != "newsapi: client is nil" {
		t.Fatalf("GetNews() error = %q, want %q", err.Error(), "newsapi: client is nil")
	}
}

func TestClientGetDoesNotConsumeRateLimitQuotaWhenURLBuildFails(t *testing.T) {
	t.Parallel()

	limiter := data.NewRateLimiter(1, time.Hour)
	client := NewClient("test-key", discardLogger(), limiter)
	client.baseURL = "://bad-url"

	_, err := client.Get(context.Background(), url.Values{
		"q": []string{"AAPL"},
	})
	if err == nil {
		t.Fatal("Get() error = nil, want non-nil")
	}
	if err.Error() == "" {
		t.Fatal("Get() error = empty, want parse base url error")
	}
	if !limiter.TryAcquire() {
		t.Fatal("TryAcquire() = false, want limiter token preserved after URL build failure")
	}
}

func TestProviderUnsupportedMethodsReturnErrNotImplemented(t *testing.T) {
	t.Parallel()

	provider := NewProvider(&Client{})

	_, ohlcvErr := provider.GetOHLCV(context.Background(), "AAPL", data.Timeframe1d, time.Now(), time.Now())
	if !errors.Is(ohlcvErr, data.ErrNotImplemented) {
		t.Fatalf("GetOHLCV() error = %v, want ErrNotImplemented", ohlcvErr)
	}

	_, fundamentalsErr := provider.GetFundamentals(context.Background(), "AAPL")
	if !errors.Is(fundamentalsErr, data.ErrNotImplemented) {
		t.Fatalf("GetFundamentals() error = %v, want ErrNotImplemented", fundamentalsErr)
	}

	_, socialErr := provider.GetSocialSentiment(context.Background(), "AAPL", time.Now().Add(-time.Hour), time.Now())
	if !errors.Is(socialErr, data.ErrNotImplemented) {
		t.Fatalf("GetSocialSentiment() error = %v, want ErrNotImplemented", socialErr)
	}
}

func TestSetTimeout_Valid(t *testing.T) {
	t.Parallel()

	client := NewClient("test-key", discardLogger())
	client.SetTimeout(30 * time.Second)

	if client.httpClient.Timeout != 30*time.Second {
		t.Fatalf("Timeout = %v, want %v", client.httpClient.Timeout, 30*time.Second)
	}
}

func TestSetTimeout_ZeroIgnored(t *testing.T) {
	t.Parallel()

	client := NewClient("test-key", discardLogger())
	original := client.httpClient.Timeout

	client.SetTimeout(0)

	if client.httpClient.Timeout != original {
		t.Fatalf("Timeout changed to %v, want original %v preserved", client.httpClient.Timeout, original)
	}
}

func TestSetTimeout_NegativeIgnored(t *testing.T) {
	t.Parallel()

	client := NewClient("test-key", discardLogger())
	original := client.httpClient.Timeout

	client.SetTimeout(-1 * time.Second)

	if client.httpClient.Timeout != original {
		t.Fatalf("Timeout changed to %v, want original %v preserved", client.httpClient.Timeout, original)
	}
}

func TestGetNews_EmptyTicker(t *testing.T) {
	t.Parallel()

	client := NewClient("test-key", discardLogger())
	provider := NewProvider(client)

	_, err := provider.GetNews(context.Background(), "", time.Time{}, time.Time{})
	if err == nil {
		t.Fatal("GetNews() error = nil, want non-nil for empty ticker")
	}
	if err.Error() != "newsapi: ticker is required" {
		t.Fatalf("GetNews() error = %q, want %q", err.Error(), "newsapi: ticker is required")
	}
}

func TestGetNews_InvalidDateRange(t *testing.T) {
	t.Parallel()

	client := NewClient("test-key", discardLogger())
	provider := NewProvider(client)

	from := time.Date(2024, time.January, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	_, err := provider.GetNews(context.Background(), "AAPL", from, to)
	if err == nil {
		t.Fatal("GetNews() error = nil, want non-nil for from > to")
	}
	if err.Error() != "newsapi: from must be before or equal to to" {
		t.Fatalf("GetNews() error = %q, want %q", err.Error(), "newsapi: from must be before or equal to to")
	}
}

func TestGetNews_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":"error","code":"unexpectedError","message":"something went wrong"}`))
	}))
	defer server.Close()

	client := NewClient("test-key", discardLogger())
	client.baseURL = server.URL
	client.httpClient = server.Client()
	provider := NewProvider(client)

	_, err := provider.GetNews(context.Background(), "AAPL", time.Time{}, time.Time{})
	if err == nil {
		t.Fatal("GetNews() error = nil, want non-nil for server 500")
	}

	var errResp *ErrorResponse
	if !errors.As(err, &errResp) {
		t.Fatalf("GetNews() error type = %T, want *ErrorResponse", err)
	}
	if errResp.StatusCode() != http.StatusInternalServerError {
		t.Fatalf("ErrorResponse.StatusCode() = %d, want %d", errResp.StatusCode(), http.StatusInternalServerError)
	}
}

func TestGetNews_MalformedJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","articles":[INVALID`))
	}))
	defer server.Close()

	client := NewClient("test-key", discardLogger())
	client.baseURL = server.URL
	client.httpClient = server.Client()
	provider := NewProvider(client)

	_, err := provider.GetNews(context.Background(), "AAPL", time.Time{}, time.Time{})
	if err == nil {
		t.Fatal("GetNews() error = nil, want non-nil for malformed JSON")
	}

	wantSubstring := "decode everything response"
	if !strings.Contains(err.Error(), wantSubstring) {
		t.Fatalf("GetNews() error = %q, want error containing %q", err.Error(), wantSubstring)
	}
}
