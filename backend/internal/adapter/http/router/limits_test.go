package router_test

import (
	"strings"
	"testing"
)

func TestLoginIsRateLimited(t *testing.T) {
	h := newServer(t)
	body := `{"username":"nobody","password":"wrong-password-1"}`
	var last int
	for i := 0; i < 12; i++ {
		rec := do(t, h, "POST", "/api/auth/login", body, nil)
		last = rec.Code
		if i < 10 && rec.Code == 429 {
			t.Fatalf("attempt %d limited too early", i+1)
		}
		if rec.Code == 429 && rec.Header().Get("Retry-After") == "" {
			t.Error("missing Retry-After")
		}
	}
	if last != 429 {
		t.Errorf("attempt 12: %d, want 429", last)
	}
}

func TestOversizedBodyIsRejected(t *testing.T) {
	h := newServer(t)
	big := `{"username":"` + strings.Repeat("a", 2<<20) + `"}`
	if rec := do(t, h, "POST", "/api/auth/login", big, nil); rec.Code != 413 && rec.Code != 400 {
		t.Errorf("oversized body: %d, want 413 or 400", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := newServer(t)
	rec := do(t, h, "GET", "/api/user", "", nil)
	for k, v := range map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
	} {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}
