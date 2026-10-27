package router

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"travel-blog/backend/internal/adapter/http/api"
	"travel-blog/backend/internal/adapter/http/auth"
)

// New wires the OpenAPI routes to the handler with plain net/http (same origin, no CORS).
// authenticate sets the user (401 without, except login and public profiles); routes adds non-JSON routes.
func New(handler api.StrictServerInterface, authenticate func(http.Handler) http.Handler, routes ...func(*http.ServeMux)) http.Handler {
	mux := http.NewServeMux()
	for _, add := range routes {
		add(mux)
	}
	strict := api.NewStrictHandlerWithOptions(handler, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
			writeProblem(w, http.StatusInternalServerError, "internal", "")
		},
	})
	api.HandlerWithOptions(strict, api.StdHTTPServerOptions{
		BaseURL:    "/api",
		BaseRouter: mux,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeProblem(w, http.StatusBadRequest, "bad_request", err.Error())
		},
	})
	return securityHeaders(noCache(limitBody(limitAuth(authenticate(requireUser(requirePasswordChange(mux)))))))
}

// securityHeaders blocks what a JSON API never needs: embedding, scripts,
// sniffing and referrers.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

var openPrefixes = []string{"/api/auth/", "/api/public/"}

func requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.UserFrom(r.Context()); !ok && !isOpen(r.URL.Path) {
			writeProblem(w, http.StatusUnauthorized, "login_required", "login required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Users who must change their password only reach their account (GET /user),
// the password change and logout (/api/auth/).
func requirePasswordChange(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth.MustChangePassword(r.Context()) && !isOpen(r.URL.Path) &&
			!(r.Method == http.MethodGet && r.URL.Path == "/api/user") &&
			!(r.Method == http.MethodPut && r.URL.Path == "/api/user/password") {
			writeProblem(w, http.StatusForbidden, "password_change_required", "password change required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isOpen(path string) bool {
	for _, p := range openPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// noCache makes the browser revalidate every response via ETag, so unchanged data costs
// only a 304. private: responses belong to one user and must stay out of shared caches.
func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-cache")
		next.ServeHTTP(w, r)
	})
}

// code is the stable error code clients translate (see docs/error-codes.md).
func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	p := api.Problem{Status: status, Title: http.StatusText(status), Code: &code}
	if detail != "" {
		p.Detail = &detail
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(p)
}
