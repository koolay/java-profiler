package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

type AuthConfig struct {
	CollectorToken string
	UIToken        string
	RequireTLS     bool
}

func BearerToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return ""
	}
	return strings.TrimSpace(auth[len("bearer "):])
}

func tokenMatches(got, want string) bool {
	if want == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func RequireCollectorAuth(cfg AuthConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.RequireTLS && r.TLS == nil {
			http.Error(w, "tls required", http.StatusUpgradeRequired)
			return
		}
		if !tokenMatches(BearerToken(r), cfg.CollectorToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireUIAuth(cfg AuthConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.RequireTLS && r.TLS == nil {
			http.Error(w, "tls required", http.StatusUpgradeRequired)
			return
		}
		if !tokenMatches(BearerToken(r), cfg.UIToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
