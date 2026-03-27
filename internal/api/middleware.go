package api

import (
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// CORSConfig holds settings for the CORS middleware.
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int // seconds
}

// DefaultCORSConfig returns a permissive CORS configuration suitable for local development.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
		MaxAge:         86400,
	}
}

// CORS returns middleware that sets Cross-Origin Resource Sharing headers.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	origins := strings.Join(cfg.AllowedOrigins, ", ")
	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")
	maxAgeStr := strconv.Itoa(cfg.MaxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origins)
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			w.Header().Set("Access-Control-Max-Age", maxAgeStr)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter provides simple per-client rate limiting based on a sliding
// window counter keyed by the client's IP address.
type RateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientWindow
	limit    int
	window   time.Duration
	nowFunc  func() time.Time
}

type clientWindow struct {
	count    int
	resetAt  time.Time
}

// NewRateLimiter creates a rate limiter that allows limit requests per window
// per client IP.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*clientWindow),
		limit:   limit,
		window:  window,
		nowFunc: time.Now,
	}
}

// Allow returns true if the client identified by key has not exceeded the rate
// limit.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.nowFunc()
	cw, ok := rl.clients[key]
	if !ok || now.After(cw.resetAt) {
		rl.clients[key] = &clientWindow{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	cw.count++
	return cw.count <= rl.limit
}

// Middleware returns an http.Handler middleware that rejects requests exceeding
// the rate limit with 429 Too Many Requests.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientIP(r)
		if !rl.Allow(key) {
			respondError(w, http.StatusTooManyRequests, "rate limit exceeded", ErrCodeRateLimited)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the client IP from the request, preferring
// X-Forwarded-For and X-Real-IP headers.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if parts := strings.SplitN(xff, ",", 2); len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RequestLogger returns middleware that logs each HTTP request using the
// provided structured logger.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusCapture{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			logger.Info("http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
		})
	}
}

// statusCapture wraps http.ResponseWriter to record the status code.
type statusCapture struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (sw *statusCapture) WriteHeader(code int) {
	if !sw.wroteHeader {
		sw.status = code
		sw.wroteHeader = true
		sw.ResponseWriter.WriteHeader(code)
	}
}

func (sw *statusCapture) Write(b []byte) (int, error) {
	if !sw.wroteHeader {
		sw.WriteHeader(http.StatusOK)
	}
	return sw.ResponseWriter.Write(b)
}
