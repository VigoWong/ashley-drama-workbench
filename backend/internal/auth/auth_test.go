package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginRejectsBadCredentials(t *testing.T) {
	a := New()
	r := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
	w := httptest.NewRecorder()
	a.LoginHandler(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLoginAcceptsDefaultsAndTokenWorks(t *testing.T) {
	a := New() // defaults admin/admin
	r := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader(`{"username":"admin","password":"admin"}`))
	w := httptest.NewRecorder()
	a.LoginHandler(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"token"`) {
		t.Fatalf("expected token in body, got %s", w.Body.String())
	}

	// The minted token must pass the middleware; a bogus token must not.
	guarded := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ok := httptest.NewRequest(http.MethodPost, "/api/generate", nil)
	ok.Header.Set("Authorization", "Bearer "+a.token)
	okW := httptest.NewRecorder()
	guarded.ServeHTTP(okW, ok)
	if okW.Code != http.StatusOK {
		t.Fatalf("valid token rejected: %d", okW.Code)
	}

	bad := httptest.NewRequest(http.MethodPost, "/api/generate", nil)
	bad.Header.Set("Authorization", "Bearer nope")
	badW := httptest.NewRecorder()
	guarded.ServeHTTP(badW, bad)
	if badW.Code != http.StatusUnauthorized {
		t.Fatalf("bad token accepted: %d", badW.Code)
	}
}
