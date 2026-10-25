// Package auth resolves the user of a request from the session cookie and stores it
// in the context (read via UserFrom).
package auth

import (
	"context"
	"log"
	"net/http"
	"time"

	"travel-blog/backend/internal/domain"
)

const CookieName = "session"

type Authenticator interface {
	Authenticate(token string) (domain.UserID, bool, error)
}

// PasswordGate tells whether a user must change their password
// before using the application (service.AccountService).
type PasswordGate interface {
	MustChangePassword(user domain.UserID) bool
}

type userKey struct{}
type tokenKey struct{}
type mustChangeKey struct{}

func WithUser(ctx context.Context, user domain.UserID) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}

func UserFrom(ctx context.Context) (user domain.UserID, ok bool) {
	user, ok = ctx.Value(userKey{}).(domain.UserID)
	return user, ok && user != ""
}

func MustChangePassword(ctx context.Context) bool {
	must, _ := ctx.Value(mustChangeKey{}).(bool)
	return must
}

// TokenFrom returns the valid session token of the request or "".
func TokenFrom(ctx context.Context) string {
	token, _ := ctx.Value(tokenKey{}).(string)
	return token
}

// Sessions stores the user of the session cookie in the context. Without a valid
// session the request stays anonymous, or runs as fallback if one is given (development only).
func Sessions(a Authenticator, fallback ...domain.UserID) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if c, err := r.Cookie(CookieName); err == nil {
				user, ok, err := a.Authenticate(c.Value)
				if err != nil {
					log.Printf("authenticate: %v", err)
				}
				if ok {
					ctx = context.WithValue(WithUser(ctx, user), tokenKey{}, c.Value)
					if gate, isGate := a.(PasswordGate); isGate && gate.MustChangePassword(user) {
						ctx = context.WithValue(ctx, mustChangeKey{}, true)
					}
				}
			}
			if _, ok := UserFrom(ctx); !ok && len(fallback) > 0 {
				ctx = WithUser(ctx, fallback[0])
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SessionCookie sets the session cookie. Secure also works on http://localhost,
// because browsers treat localhost as a secure origin.
func SessionCookie(token string, ttl time.Duration) string {
	return (&http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/api",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}).String()
}

func ClearCookie() string {
	return (&http.Cookie{
		Name:     CookieName,
		Path:     "/api",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}).String()
}
