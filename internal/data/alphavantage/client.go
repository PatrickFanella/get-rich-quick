package alphavantage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
)

const (
	defaultBaseURL = "https://www.alphavantage.co/query"
	defaultTimeout = 30 * time.Second
)

// Client is a small HTTP client for Alpha Vantage APIs.
type Client struct {
	apiKey       string
	baseURL      string
	httpClient   *http.Client
	api          *data.APIClient
	logger       *slog.Logger
	rateLimiters []*data.RateLimiter
}

// ErrorResponse captures Alpha Vantage's standard error response shapes.
type ErrorResponse struct {
	Information  string `json:"Information"`
	Note         string `json:"Note"`
	ErrorMessage string `json:"Error Message"`
	Message      string `json:"message"`

	statusCode int
}

// NewClient constructs an Alpha Vantage HTTP client.
// If logger is nil, slog.Default() is used.
func NewClient(apiKey string, logger *slog.Logger, rateLimiters ...*data.RateLimiter) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	trimmedKey := strings.TrimSpace(apiKey)
	httpClient := &http.Client{
		Timeout: defaultTimeout,
	}

	api := data.NewAPIClient(data.APIClientConfig{
		BaseURL: defaultBaseURL,
		Auth: data.AuthConfig{
			Style:     data.AuthStyleQueryParam,
			ParamName: "apikey",
			Value:     trimmedKey,
		},
		Timeout: defaultTimeout,
		Logger:  logger,
		Prefix:  "alphavantage",
	})
	api.SetHTTPClient(httpClient)

	client := &Client{
		apiKey:     trimmedKey,
		baseURL:    defaultBaseURL,
		httpClient: httpClient,
		api:        api,
		logger:     logger,
	}

	for _, limiter := range rateLimiters {
		if limiter != nil {
			client.rateLimiters = append(client.rateLimiters, limiter)
		}
	}

	return client
}

// SetTimeout updates the timeout used by the underlying HTTP client.
func (c *Client) SetTimeout(timeout time.Duration) {
	if c == nil {
		return
	}
	if timeout <= 0 {
		c.logger.Warn("alphavantage: ignoring invalid timeout", slog.String("timeout", timeout.String()))
		return
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{}
	}

	c.httpClient.Timeout = timeout
}

// Get issues a GET request to the Alpha Vantage query endpoint and returns the raw response body.
func (c *Client) Get(ctx context.Context, params url.Values) ([]byte, error) {
	if c == nil {
		return nil, errors.New("alphavantage: client is nil")
	}
	if c.apiKey == "" {
		return nil, errors.New("alphavantage: api key is required")
	}

	// Sync baseURL in case tests changed it directly.
	if c.baseURL != c.api.BaseURL() {
		c.api.SetBaseURL(c.baseURL)
	}

	reservations, err := c.reserveRateLimiters(ctx)
	if err != nil {
		return nil, fmt.Errorf("alphavantage: wait for rate limiter: %w", err)
	}
	committedReservations := false
	defer func() {
		if !committedReservations {
			cancelReservations(reservations)
		}
	}()

	body, _, err := c.api.Get(ctx, "", params)
	if err != nil {
		var apiErr *data.APIError
		if errors.As(err, &apiErr) {
			// Commit reservations on successful HTTP round-trip
			// (the request was made, even though the server returned an error).
			commitReservations(reservations)
			committedReservations = true

			if avErr := parseErrorResponse(apiErr.StatusCode, apiErr.Body); avErr != nil {
				c.logger.Warn("alphavantage: non-success response",
					slog.Int("status", avErr.StatusCode()),
					slog.Any("error", avErr),
				)
				return nil, avErr
			}
		}
		return nil, err
	}

	// Commit reservations on successful response.
	commitReservations(reservations)
	committedReservations = true

	// Alpha Vantage may return errors inside 200 OK responses.
	if avErr := parseErrorResponse(http.StatusOK, body); avErr != nil {
		c.logger.Warn("alphavantage: non-success response",
			slog.Int("status", avErr.StatusCode()),
			slog.Any("error", avErr),
		)
		return nil, avErr
	}

	return body, nil
}

// StatusCode returns the HTTP status code for the error response.
func (e *ErrorResponse) StatusCode() int {
	if e == nil {
		return 0
	}
	return e.statusCode
}

func (e *ErrorResponse) Error() string {
	if e == nil {
		return "alphavantage: request failed"
	}

	message := strings.TrimSpace(e.ErrorMessage)
	if message == "" {
		message = strings.TrimSpace(e.Note)
	}
	if message == "" {
		message = strings.TrimSpace(e.Information)
	}
	if message == "" {
		message = strings.TrimSpace(e.Message)
	}
	if message == "" {
		message = http.StatusText(e.statusCode)
	}
	if message == "" {
		message = "request failed"
	}

	return fmt.Sprintf("alphavantage: %s (status=%d)", message, e.statusCode)
}

func (c *Client) reserveRateLimiters(ctx context.Context) ([]*data.Reservation, error) {
	reservations := make([]*data.Reservation, 0, len(c.rateLimiters))
	for _, limiter := range c.rateLimiters {
		if limiter == nil {
			continue
		}
		reservation, err := limiter.Reserve(ctx)
		if err != nil {
			cancelReservations(reservations)
			return nil, err
		}
		reservations = append(reservations, reservation)
	}

	return reservations, nil
}

func parseErrorResponse(statusCode int, body []byte) *ErrorResponse {
	if isSuccessStatusCode(statusCode) && len(body) == 0 {
		return nil
	}

	errResp := &ErrorResponse{statusCode: statusCode}
	if len(body) > 0 {
		if err := json.Unmarshal(body, errResp); err != nil {
			if !isSuccessStatusCode(statusCode) {
				errResp.Message = strings.TrimSpace(string(body))
			} else {
				return nil
			}
		}
	}

	if isSuccessStatusCode(statusCode) && !errResp.hasErrorMessage() {
		return nil
	}

	if isSuccessStatusCode(statusCode) {
		errResp.statusCode = errResp.syntheticStatusCode()
	}

	if !errResp.hasErrorMessage() && errResp.Message == "" {
		errResp.Message = strings.TrimSpace(string(body))
	}

	return errResp
}

func (e *ErrorResponse) hasErrorMessage() bool {
	if e == nil {
		return false
	}

	return strings.TrimSpace(e.ErrorMessage) != "" ||
		strings.TrimSpace(e.Note) != "" ||
		strings.TrimSpace(e.Information) != "" ||
		strings.TrimSpace(e.Message) != ""
}

func (e *ErrorResponse) syntheticStatusCode() int {
	if e == nil {
		return http.StatusBadGateway
	}

	if strings.TrimSpace(e.Note) != "" {
		return http.StatusTooManyRequests
	}

	message := strings.ToLower(strings.TrimSpace(strings.Join([]string{
		e.ErrorMessage,
		e.Information,
		e.Message,
	}, " ")))
	if strings.Contains(message, "api key") || strings.Contains(message, "apikey") {
		return http.StatusUnauthorized
	}

	return http.StatusBadRequest
}

func isSuccessStatusCode(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
}

func commitReservations(reservations []*data.Reservation) {
	for _, reservation := range reservations {
		if reservation == nil {
			continue
		}
		reservation.Commit()
	}
}

func cancelReservations(reservations []*data.Reservation) {
	for i := len(reservations) - 1; i >= 0; i-- {
		if reservations[i] == nil {
			continue
		}
		reservations[i].Cancel()
	}
}
