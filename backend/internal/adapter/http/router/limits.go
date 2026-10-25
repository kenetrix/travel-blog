package router

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// maxJSONBody limits requests with JSON and all other bodies.
	maxJSONBody = 1 << 20
	// maxUploadBody limits photo uploads (photo up to 20 MB plus form fields).
	maxUploadBody = 21 << 20

	// loginAttempts attempts are allowed per address per loginWindow.
	loginAttempts = 10
	loginWindow   = time.Minute
)

func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := int64(maxJSONBody)
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
			limit = maxUploadBody
		}
		if r.ContentLength > limit {
			writeProblem(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}

type attemptLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	entries map[string]*attempts
}

type attempts struct {
	n     int
	reset time.Time
}

func newAttemptLimiter() *attemptLimiter {
	return &attemptLimiter{now: time.Now, entries: map[string]*attempts{}}
}

// allow counts one attempt; at the limit it returns false and the wait
// until the next window.
func (l *attemptLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for k, e := range l.entries {
		if now.After(e.reset) {
			delete(l.entries, k)
		}
	}
	e := l.entries[key]
	if e == nil {
		e = &attempts{reset: now.Add(loginWindow)}
		l.entries[key] = e
	}
	if e.n >= loginAttempts {
		return false, e.reset.Sub(now)
	}
	e.n++
	return true, 0
}

// limitAuth throttles login and registration against guessing. The address comes
// from RemoteAddr; X-Forwarded-For is not trusted because clients can set it freely.
func limitAuth(next http.Handler) http.Handler {
	l := newAttemptLimiter()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost &&
			(r.URL.Path == "/api/auth/login" || r.URL.Path == "/api/auth/register") {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			if ok, wait := l.allow(host); !ok {
				w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
				writeProblem(w, http.StatusTooManyRequests, "too many attempts, try again later")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
