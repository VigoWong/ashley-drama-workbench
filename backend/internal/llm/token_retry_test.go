package llm

import (
	"errors"
	"testing"

	"golang.org/x/oauth2"
)

// flakyTokenSource fails its first `failures` calls, then succeeds — modelling a
// proxy node that intermittently drops the connection to oauth2.googleapis.com
// ("unexpected EOF") before a retry lands on a healthy path.
type flakyTokenSource struct {
	failures int
	calls    int
	err      error
}

func (f *flakyTokenSource) Token() (*oauth2.Token, error) {
	f.calls++
	if f.calls <= f.failures {
		return nil, f.err
	}
	return &oauth2.Token{AccessToken: "ok"}, nil
}

// A transient EOF on the first attempt(s) must be absorbed by the retry, so the
// chat path (which calls doGenerate directly, without GenerateJSON's retry) no
// longer surfaces "vertex token: ... unexpected EOF" to the user.
func TestRetryTokenSourceRecoversFromTransientFailures(t *testing.T) {
	flaky := &flakyTokenSource{failures: 2, err: errors.New("unexpected EOF")}
	rts := retryTokenSource{base: flaky, attempts: 3, backoff: 0}

	tok, err := rts.Token()
	if err != nil {
		t.Fatalf("expected success after retries, got %v (calls=%d)", err, flaky.calls)
	}
	if tok.AccessToken != "ok" {
		t.Fatalf("unexpected token %q", tok.AccessToken)
	}
	if flaky.calls != 3 {
		t.Fatalf("expected 3 calls (2 fail + 1 ok), got %d", flaky.calls)
	}
}

// A genuinely broken token source (bad creds) must still surface an error after
// exhausting the attempts — retry adds resilience, it does not hide hard faults.
func TestRetryTokenSourceGivesUpAfterMaxAttempts(t *testing.T) {
	flaky := &flakyTokenSource{failures: 99, err: errors.New("invalid_grant")}
	rts := retryTokenSource{base: flaky, attempts: 3, backoff: 0}

	_, err := rts.Token()
	if err == nil {
		t.Fatal("expected error after exhausting attempts")
	}
	if flaky.calls != 3 {
		t.Fatalf("expected exactly 3 attempts, got %d", flaky.calls)
	}
}
