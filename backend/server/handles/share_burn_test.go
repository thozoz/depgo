package handles_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/alist-org/alist/v3/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createLimitedShare creates a share whose access budget is capped at limit and
// returns its id plus a valid access token.
func createLimitedShare(t *testing.T, handler http.Handler, adminToken, rootDir, filename, password string, body string) (shareID, token string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, filename), []byte("BURN-SECRET-CONTENT"), 0644))

	req := httptest.NewRequest("POST", "/api/share/create", strings.NewReader(body))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "share create failed: %s", w.Body.String())

	var created struct {
		Data struct {
			ShareID string `json:"share_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	shareID = created.Data.ShareID
	require.NotEmpty(t, shareID)

	authBody := fmt.Sprintf(`{"share_id":"%s","password":"%s"}`, shareID, password)
	reqAuth := httptest.NewRequest("POST", "/api/public/share/auth", strings.NewReader(authBody))
	reqAuth.Header.Set("Content-Type", "application/json")
	wAuth := httptest.NewRecorder()
	handler.ServeHTTP(wAuth, reqAuth)
	require.Equal(t, 200, parseRespCode(t, wAuth), "share auth failed: %s", wAuth.Body.String())

	var authed struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wAuth.Body.Bytes(), &authed))
	token = authed.Data.Token
	require.NotEmpty(t, token)
	return shareID, token
}

// TestBurnAfterReadServesExactlyOneRecipient is the regression test for an
// over-serve race: eligibility is checked before the content is resolved, so
// concurrent requests all observe "one access left" and all get served, turning
// a burn-after-read link into an unlimited one.
func TestBurnAfterReadServesExactlyOneRecipient(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, token := createLimitedShare(t, r, adminToken, rootDir, "burn.txt", "burnpass",
		`{"path":"/burn.txt","name":"burn","password":"burnpass","burn_after_read":true}`)

	const concurrency = 8
	var wg sync.WaitGroup
	bodies := make([]string, concurrency)
	statuses := make([]int, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", fmt.Sprintf("/sd/%s?auth=%s", shareID, token), nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			bodies[i] = w.Body.String()
			statuses[i] = w.Code
		}(i)
	}
	wg.Wait()

	served := 0
	for i := range bodies {
		if strings.Contains(bodies[i], "BURN-SECRET-CONTENT") {
			served++
		}
	}
	assert.Equal(t, 1, served,
		"a burn-after-read share must serve exactly one recipient, served=%d/8 statuses=%v", served, statuses)

	after, err := db.GetShareByShareID(shareID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), after.AccessCount, "access_count must stop at the limit")
	assert.True(t, after.IsConsumed(), "share must be consumed")
}

// TestAccessLimitIsEnforcedUnderConcurrency covers the non-burn variant: an
// explicit access_limit of 2 must serve two recipients, not more.
func TestAccessLimitIsEnforcedUnderConcurrency(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, token := createLimitedShare(t, r, adminToken, rootDir, "limit2.txt", "limitpass",
		`{"path":"/limit2.txt","name":"limit2","password":"limitpass","access_limit":2}`)

	const concurrency = 8
	var wg sync.WaitGroup
	bodies := make([]string, concurrency)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", fmt.Sprintf("/sd/%s?auth=%s", shareID, token), nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			bodies[i] = w.Body.String()
		}(i)
	}
	wg.Wait()

	served := 0
	for i := range bodies {
		if strings.Contains(bodies[i], "BURN-SECRET-CONTENT") {
			served++
		}
	}
	assert.Equal(t, 2, served, "access_limit=2 must serve exactly two recipients, served=%d", served)

	after, err := db.GetShareByShareID(shareID)
	require.NoError(t, err)
	assert.LessOrEqual(t, after.AccessCount, int64(2), "access_count must never exceed the limit")
}

// TestConsumedShareIsRefusedAfterBudgetSpent checks the plain sequential case
// still answers 410 once the budget is gone.
func TestConsumedShareIsRefusedAfterBudgetSpent(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, token := createLimitedShare(t, r, adminToken, rootDir, "once.txt", "oncepass",
		`{"path":"/once.txt","name":"once","password":"oncepass","burn_after_read":true}`)

	first := httptest.NewRecorder()
	r.ServeHTTP(first, httptest.NewRequest("GET", fmt.Sprintf("/sd/%s?auth=%s", shareID, token), nil))
	assert.Contains(t, first.Body.String(), "BURN-SECRET-CONTENT", "the first request must be served")

	second := httptest.NewRecorder()
	r.ServeHTTP(second, httptest.NewRequest("GET", fmt.Sprintf("/sd/%s?auth=%s", shareID, token), nil))
	_, bodyCode, msg := parseRespEnvelope(t, second)
	assert.NotEqual(t, 200, bodyCode, "a consumed share must not be served again, msg=%q", msg)
	assert.NotContains(t, second.Body.String(), "BURN-SECRET-CONTENT")
}
