package llm

import (
	"time"

	"golang.org/x/oauth2"
)

// retryTokenSource wraps an oauth2.TokenSource and retries transient failures
// when minting a Vertex access token. In the NAS deployment the token exchange
// to oauth2.googleapis.com goes through a proxy whose node occasionally drops
// the connection ("unexpected EOF"); a single retry reliably lands on a healthy
// path (measured ~30/30 success on retry).
//
// We retry at the *token* boundary rather than around the whole request because
// the OAuth token exchange is idempotent and cheap, whereas the generateContent
// POST it guards is not (retrying could double-bill). The text path's
// GenerateJSON already retries the whole call; this decorator protects paths
// that reach doGenerate directly (e.g. chat's GenerateWithTools).
//
// The underlying source is oauth2's caching ReuseTokenSource, so a valid token
// returns from cache with no network and no retry — the loop only re-runs on an
// actual mint that failed.
type retryTokenSource struct {
	base     oauth2.TokenSource
	attempts int
	backoff  time.Duration // base linear backoff; attempt i sleeps (i+1)*backoff
}

func (r retryTokenSource) Token() (*oauth2.Token, error) {
	var lastErr error
	for i := 0; i < r.attempts; i++ {
		tok, err := r.base.Token()
		if err == nil {
			return tok, nil
		}
		lastErr = err
		if i < r.attempts-1 {
			time.Sleep(time.Duration(i+1) * r.backoff)
		}
	}
	return nil, lastErr
}
