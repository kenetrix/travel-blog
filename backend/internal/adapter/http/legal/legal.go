// Package legal serves the operator's contact details for the Impressum. They come from
// the environment, so no personal data is part of the public repository.
package legal

import (
	"encoding/json"
	"net/http"
	"strings"
)

type notice struct {
	Name    string   `json:"name"`
	Address []string `json:"address"`
	Email   string   `json:"email"`
}

// fromEnv reads LEGAL_NAME, LEGAL_ADDRESS (lines separated by "|") and LEGAL_EMAIL.
func fromEnv(getenv func(string) string) notice {
	n := notice{
		Name:    strings.TrimSpace(getenv("LEGAL_NAME")),
		Address: []string{},
		Email:   strings.TrimSpace(getenv("LEGAL_EMAIL")),
	}
	for _, line := range strings.Split(getenv("LEGAL_ADDRESS"), "|") {
		if line = strings.TrimSpace(line); line != "" {
			n.Address = append(n.Address, line)
		}
	}
	return n
}

// Routes serves GET /api/public/legal, outside the OpenAPI contract like the OIDC flow.
func Routes(getenv func(string) string) func(*http.ServeMux) {
	n := fromEnv(getenv)
	return func(mux *http.ServeMux) {
		mux.HandleFunc("GET /api/public/legal", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(n)
		})
	}
}
