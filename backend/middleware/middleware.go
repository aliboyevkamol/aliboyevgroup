// Package middleware: request id, structured logging, recovery, security headers, CORS and rate limiting.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ctxKey int

const ridKey ctxKey = 1

func RID(r *http.Request) string { v, _ := r.Context().Value(ridKey).(string); return v }

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		id := hex.EncodeToString(b)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ridKey, id)))
	})
}

type rec struct {
	http.ResponseWriter
	status int
}

func (r *rec) WriteHeader(c int) { r.status = c; r.ResponseWriter.WriteHeader(c) }

// Logger logs method, path (never the query string or bodies), status, duration and request id.
func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t := time.Now()
			rw := &rec{ResponseWriter: w, status: 200}
			next.ServeHTTP(rw, r)
			log.Info("request", "method", r.Method, "path", r.URL.Path, "status", rw.status, "duration_ms", time.Since(t).Milliseconds(), "request_id", RID(r))
		})
	}
}

func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					log.Error("panic", "value", v, "path", r.URL.Path, "request_id", RID(r))
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(500)
					_, _ = w.Write([]byte(`{"success":false,"message":"Something went wrong. Please try again.","error":{"code":"INTERNAL_ERROR"}}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func Security(prod bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			h.Set("X-Frame-Options", "SAMEORIGIN")
			if strings.HasPrefix(r.URL.Path, "/api/") {
				h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
				h.Set("Cache-Control", "no-store")
				h.Set("X-Frame-Options", "DENY")
			}
			if prod {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			o := r.Header.Get("Origin")
			if o != "" && allowed[o] {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", o)
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Vary", "Origin")
				if r.Method == http.MethodOptions {
					h.Set("Access-Control-Allow-Methods", "GET,POST,PATCH,PUT,DELETE,OPTIONS")
					h.Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Idempotency-Key,X-Requested-With")
					h.Set("Access-Control-Max-Age", "600")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

type bucket struct {
	tokens float64
	last   time.Time
}

// Limiter is an in-memory token bucket per client IP. Use one instance per policy.
type Limiter struct {
	mu         sync.Mutex
	m          map[string]*bucket
	perMinute  float64
	burst      float64
	trustProxy bool
}

func NewLimiter(perMinute, burst int, trustProxy bool) *Limiter {
	l := &Limiter{m: map[string]*bucket{}, perMinute: float64(perMinute), burst: float64(burst), trustProxy: trustProxy}
	go func() {
		for range time.Tick(5 * time.Minute) {
			l.mu.Lock()
			for k, b := range l.m {
				if time.Since(b.last) > 10*time.Minute {
					delete(l.m, k)
				}
			}
			l.mu.Unlock()
		}
	}()
	return l
}

func (l *Limiter) ip(r *http.Request) string {
	if l.trustProxy {
		if f := r.Header.Get("X-Forwarded-For"); f != "" {
			return strings.TrimSpace(strings.Split(f, ",")[0])
		}
	}
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return h
}

func (l *Limiter) Allow(r *http.Request) bool {
	k := l.ip(r)
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.m[k]
	now := time.Now()
	if b == nil {
		b = &bucket{tokens: l.burst, last: now}
		l.m[k] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.perMinute / 60
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.Allow(r) {
			w.Header().Set("Retry-After", "30")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(429)
			_, _ = w.Write([]byte(`{"success":false,"message":"Too many requests. Please slow down.","error":{"code":"RATE_LIMITED"}}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
