// Package auth provides a minimal single-user token gate for the workbench.
// Credentials default to admin/admin and can be overridden via env
// (AUTH_USERNAME / AUTH_PASSWORD). On successful login the server hands out an
// in-memory session token that the generate endpoint requires as a Bearer token.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

type Authenticator struct {
	user  string
	pass  string
	token string
}

// New builds an Authenticator, reading AUTH_USERNAME/AUTH_PASSWORD (default
// admin/admin) and minting a random session token that lives for the process.
func New() *Authenticator {
	user := envOr("AUTH_USERNAME", "admin")
	pass := envOr("AUTH_PASSWORD", "admin")
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("auth: cannot read random: " + err.Error())
	}
	return &Authenticator{user: user, pass: pass, token: hex.EncodeToString(b)}
}

// User returns the configured username (for logging).
func (a *Authenticator) User() string { return a.user }

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginHandler validates credentials and returns {"token": "..."} on success.
func (a *Authenticator) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}
	userOK := subtle.ConstantTimeCompare([]byte(req.Username), []byte(a.user)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(req.Password), []byte(a.pass)) == 1
	if !userOK || !passOK {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "用户名或密码错误"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": a.token})
}

// Middleware rejects requests without a valid Bearer token.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if tok == "" || subtle.ConstantTimeCompare([]byte(tok), []byte(a.token)) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "未授权,请先登录"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
