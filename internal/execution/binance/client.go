package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	testnetBaseURL      = "https://testnet.binance.vision"
	productionBaseURL   = "https://api.binance.com"
	defaultTimeout      = 30 * time.Second
	defaultRecvWindow   = 5 * time.Second
	binanceAPIKeyHeader = "X-MBX-APIKEY"
)

// Client is a small HTTP client for Binance Spot REST endpoints.
type Client struct {
	apiKey     string
	apiSecret  string
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	now        func() time.Time
	recvWindow time.Duration
}

// ErrorResponse captures Binance's standard error response shape.
type ErrorResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`

	statusCode int
}

// NewClient constructs a Binance HTTP client.
// If logger is nil, slog.Default() is used.
func NewClient(apiKey, apiSecret string, isTestnet bool, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	baseURL := productionBaseURL
	if isTestnet {
		baseURL = testnetBaseURL
	}

	return &Client{
		apiKey:    strings.TrimSpace(apiKey),
		apiSecret: strings.TrimSpace(apiSecret),
		baseURL:   baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		logger:     logger,
		now:        time.Now,
		recvWindow: defaultRecvWindow,
	}
}

// StatusCode returns the HTTP status code associated with the error.
func (e *ErrorResponse) StatusCode() int {
	if e == nil {
		return 0
	}

	return e.statusCode
}

func (e *ErrorResponse) Error() string {
	if e == nil {
		return "binance: request failed"
	}

	message := strings.TrimSpace(e.Msg)
	if message == "" {
		message = http.StatusText(e.statusCode)
	}
	if message == "" {
		message = "request failed"
	}

	return fmt.Sprintf("binance: %s (status=%d)", message, e.statusCode)
}

// SetBaseURL overrides the configured base URL. This is primarily useful for testing.
func (c *Client) SetBaseURL(baseURL string) {
	if c == nil {
		return
	}

	c.baseURL = baseURL
}

// SetHTTPClient replaces the underlying HTTP client. This is primarily useful for testing.
func (c *Client) SetHTTPClient(httpClient *http.Client) {
	if c == nil || httpClient == nil {
		return
	}

	c.httpClient = httpClient
}

// SetTimeout updates the timeout used by the underlying HTTP client.
func (c *Client) SetTimeout(timeout time.Duration) {
	if c == nil {
		return
	}

	logger := c.getLogger()
	if timeout <= 0 {
		logger.Warn("binance: ignoring invalid timeout", slog.String("timeout", timeout.String()))
		return
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: defaultTimeout}
	}

	c.httpClient.Timeout = timeout
}

// SetRecvWindow updates the recvWindow applied to signed requests.
func (c *Client) SetRecvWindow(recvWindow time.Duration) {
	if c == nil {
		return
	}

	logger := c.getLogger()
	if recvWindow <= 0 {
		logger.Warn("binance: ignoring invalid recvWindow", slog.String("recvWindow", recvWindow.String()))
		return
	}

	c.recvWindow = recvWindow
}

// Get issues an unsigned GET request and returns the raw response body.
func (c *Client) Get(ctx context.Context, requestPath string, params url.Values) ([]byte, error) {
	return c.do(ctx, http.MethodGet, requestPath, params, false)
}

// Post issues an unsigned POST request and returns the raw response body.
func (c *Client) Post(ctx context.Context, requestPath string, params url.Values) ([]byte, error) {
	return c.do(ctx, http.MethodPost, requestPath, params, false)
}

// Delete issues an unsigned DELETE request and returns the raw response body.
func (c *Client) Delete(ctx context.Context, requestPath string, params url.Values) ([]byte, error) {
	return c.do(ctx, http.MethodDelete, requestPath, params, false)
}

// SignedGet issues a signed GET request and returns the raw response body.
func (c *Client) SignedGet(ctx context.Context, requestPath string, params url.Values) ([]byte, error) {
	return c.do(ctx, http.MethodGet, requestPath, params, true)
}

// SignedPost issues a signed POST request and returns the raw response body.
func (c *Client) SignedPost(ctx context.Context, requestPath string, params url.Values) ([]byte, error) {
	return c.do(ctx, http.MethodPost, requestPath, params, true)
}

// SignedDelete issues a signed DELETE request and returns the raw response body.
func (c *Client) SignedDelete(ctx context.Context, requestPath string, params url.Values) ([]byte, error) {
	return c.do(ctx, http.MethodDelete, requestPath, params, true)
}

func (c *Client) do(ctx context.Context, method, requestPath string, params url.Values, signed bool) ([]byte, error) {
	if c == nil {
		return nil, errors.New("binance: client is nil")
	}
	if signed {
		if c.apiKey == "" {
			return nil, errors.New("binance: api key is required")
		}
		if c.apiSecret == "" {
			return nil, errors.New("binance: api secret is required")
		}
	}

	logger := c.getLogger()
	httpClient := c.getHTTPClient()

	requestURL, bodyReader, err := c.buildRequestTarget(method, requestPath, params, signed)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("binance: create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if signed {
		req.Header.Set(binanceAPIKeyHeader, c.apiKey)
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	startedAt := time.Now()
	logger.Debug("binance: sending request",
		slog.String("method", req.Method),
		slog.String("path", req.URL.Path),
	)

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Warn("binance: request failed",
			slog.String("method", req.Method),
			slog.String("path", req.URL.Path),
			slog.Any("error", err),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return nil, fmt.Errorf("binance: do request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Warn("binance: failed to close response body", slog.Any("error", closeErr))
		}
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("binance: read response body: %w", err)
	}

	logger.Debug("binance: received response",
		slog.String("method", req.Method),
		slog.String("path", req.URL.Path),
		slog.Int("status", resp.StatusCode),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, parseErrorResponse(resp.StatusCode, responseBody)
	}

	return responseBody, nil
}

func (c *Client) buildRequestTarget(method, requestPath string, params url.Values, signed bool) (string, io.Reader, error) {
	encodedParams, err := c.encodeParams(params, signed)
	if err != nil {
		return "", nil, err
	}

	if signed || method == http.MethodGet {
		requestURL, err := c.buildURL(requestPath, encodedParams)
		return requestURL, nil, err
	}

	requestURL, err := c.buildURL(requestPath, "")
	if err != nil {
		return "", nil, err
	}
	if encodedParams == "" {
		return requestURL, nil, nil
	}

	return requestURL, strings.NewReader(encodedParams), nil
}

func (c *Client) encodeParams(params url.Values, signed bool) (string, error) {
	prepared := cloneValues(params)
	if !signed {
		return prepared.Encode(), nil
	}

	prepared.Del("signature")
	if prepared.Get("timestamp") == "" {
		now := c.nowFunc()
		prepared.Set("timestamp", strconv.FormatInt(now().UnixMilli(), 10))
	}
	if prepared.Get("recvWindow") == "" && c.recvWindow > 0 {
		prepared.Set("recvWindow", strconv.FormatInt(c.recvWindow.Milliseconds(), 10))
	}

	payload := prepared.Encode()
	signature := c.generateSignature(payload)
	prepared.Set("signature", signature)

	return prepared.Encode(), nil
}

func (c *Client) generateSignature(payload string) string {
	mac := hmac.New(sha256.New, []byte(c.apiSecret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *Client) buildURL(requestPath, encodedQuery string) (string, error) {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("binance: parse base url: %w", err)
	}

	baseURL.Path = joinPath(baseURL.Path, requestPath)
	baseURL.RawPath = ""
	if unescapedPath, err := url.PathUnescape(baseURL.Path); err == nil && unescapedPath != baseURL.Path {
		baseURL.RawPath = baseURL.Path
		baseURL.Path = unescapedPath
	}
	baseURL.RawQuery = encodedQuery

	return baseURL.String(), nil
}

func parseErrorResponse(statusCode int, body []byte) *ErrorResponse {
	errResp := &ErrorResponse{statusCode: statusCode}
	if len(body) == 0 {
		return errResp
	}

	if err := json.Unmarshal(body, errResp); err != nil {
		errResp.Msg = strings.TrimSpace(string(body))
	}
	if strings.TrimSpace(errResp.Msg) == "" {
		errResp.Msg = strings.TrimSpace(string(body))
	}

	return errResp
}

func joinPath(basePath, requestPath string) string {
	trimmedPath := strings.TrimSpace(requestPath)
	if trimmedPath == "" {
		if basePath == "" {
			return "/"
		}
		return basePath
	}

	cleanPath := "/" + strings.TrimLeft(trimmedPath, "/")
	if basePath == "" || basePath == "/" {
		return cleanPath
	}

	return strings.TrimRight(basePath, "/") + cleanPath
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return url.Values{}
	}

	cloned := make(url.Values, len(values))
	for key, entries := range values {
		cloned[key] = append([]string(nil), entries...)
	}

	return cloned
}

func (c *Client) getLogger() *slog.Logger {
	if c == nil || c.logger == nil {
		return slog.Default()
	}

	return c.logger
}

func (c *Client) getHTTPClient() *http.Client {
	if c == nil || c.httpClient == nil {
		return &http.Client{Timeout: defaultTimeout}
	}

	return c.httpClient
}

func (c *Client) nowFunc() func() time.Time {
	if c == nil || c.now == nil {
		return time.Now
	}

	return c.now
}
