package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewClient_SelectsBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		isTestnet bool
		expected  string
	}{
		{
			name:      "testnet trading",
			isTestnet: true,
			expected:  testnetBaseURL,
		},
		{
			name:      "production trading",
			isTestnet: false,
			expected:  productionBaseURL,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := NewClient("key", "secret", tt.isTestnet, discardLogger())
			if client.baseURL != tt.expected {
				t.Fatalf("baseURL = %q, want %q", client.baseURL, tt.expected)
			}
		})
	}
}

func TestClientGenerateSignature_KnownInput(t *testing.T) {
	t.Parallel()

	client := NewClient(
		"test-key",
		"NhqPtmdSJYdKjVHjA7PZj4Mge3R5YNiP1e3UZjInClVN65XAbvqqM6A7H5fATj0j",
		true,
		discardLogger(),
	)

	payload := "symbol=LTCBTC&side=BUY&type=LIMIT&timeInForce=GTC&quantity=1&price=0.1&recvWindow=5000&timestamp=1499827319559"
	got := client.generateSignature(payload)
	want := "c8db56825ae71d6d79447849e617115f4a920fa2acdcab2b053c4b2838bd6b71"
	if got != want {
		t.Fatalf("generateSignature() = %q, want %q", got, want)
	}
}

func TestClientSignedPost_AddsAuthDefaultsAndSignature(t *testing.T) {
	t.Parallel()

	type requestDetails struct {
		method      string
		path        string
		contentType string
		apiKey      string
		body        url.Values
	}

	requests := make(chan requestDetails, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		requests <- requestDetails{
			method:      r.Method,
			path:        r.URL.Path,
			contentType: r.Header.Get("Content-Type"),
			apiKey:      r.Header.Get(binanceAPIKeyHeader),
			body:        mustParseQuery(t, string(payload)),
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := NewClient("test-key", "test-secret", true, discardLogger())
	client.SetBaseURL(server.URL)
	client.now = func() time.Time {
		return time.UnixMilli(1700000000123)
	}
	client.SetRecvWindow(10 * time.Second)

	body, err := client.SignedPost(context.Background(), "/api/v3/order", url.Values{
		"symbol": []string{"BTCUSDT"},
		"side":   []string{"BUY"},
		"type":   []string{"MARKET"},
	})
	if err != nil {
		t.Fatalf("SignedPost() error = %v", err)
	}
	if got := string(body); got != `{"status":"ok"}` {
		t.Fatalf("SignedPost() body = %q, want %q", got, `{"status":"ok"}`)
	}

	select {
	case request := <-requests:
		if request.method != http.MethodPost {
			t.Fatalf("request method = %s, want %s", request.method, http.MethodPost)
		}
		if request.path != "/api/v3/order" {
			t.Fatalf("request path = %s, want %s", request.path, "/api/v3/order")
		}
		if request.contentType != "application/x-www-form-urlencoded" {
			t.Fatalf("Content-Type = %q, want %q", request.contentType, "application/x-www-form-urlencoded")
		}
		if request.apiKey != "test-key" {
			t.Fatalf("%s = %q, want %q", binanceAPIKeyHeader, request.apiKey, "test-key")
		}
		if request.body.Get("symbol") != "BTCUSDT" {
			t.Fatalf("symbol = %q, want %q", request.body.Get("symbol"), "BTCUSDT")
		}
		if request.body.Get("side") != "BUY" {
			t.Fatalf("side = %q, want %q", request.body.Get("side"), "BUY")
		}
		if request.body.Get("type") != "MARKET" {
			t.Fatalf("type = %q, want %q", request.body.Get("type"), "MARKET")
		}
		if request.body.Get("timestamp") != "1700000000123" {
			t.Fatalf("timestamp = %q, want %q", request.body.Get("timestamp"), "1700000000123")
		}
		if request.body.Get("recvWindow") != "10000" {
			t.Fatalf("recvWindow = %q, want %q", request.body.Get("recvWindow"), "10000")
		}

		unsigned := cloneValues(request.body)
		unsigned.Del("signature")

		wantSignature := computeSignature("test-secret", unsigned.Encode())
		if request.body.Get("signature") != wantSignature {
			t.Fatalf("signature = %q, want %q", request.body.Get("signature"), wantSignature)
		}
	case <-time.After(time.Second):
		t.Fatal("request details were not captured")
	}
}

func TestClientSignedGet_RejectsMissingCredentials(t *testing.T) {
	t.Parallel()

	client := NewClient("", "", true, discardLogger())

	_, err := client.SignedGet(context.Background(), "/api/v3/account", nil)
	if err == nil {
		t.Fatal("SignedGet() error = nil, want non-nil")
	}
	if err.Error() != "binance: api key is required" {
		t.Fatalf("SignedGet() error = %v, want api key validation", err)
	}
}

func mustParseQuery(t *testing.T, raw string) url.Values {
	t.Helper()

	values, err := url.ParseQuery(strings.TrimSpace(raw))
	if err != nil {
		t.Fatalf("ParseQuery() error = %v", err)
	}

	return values
}

func computeSignature(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
