package handles_test

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// While a share is off its slug stays taken; reusing it fails with the same
// message every time, and enabling brings the same link back.
func TestDisabledShareKeepsItsSlugAndCanBeEnabledAgain(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, _ := createShareFixture(t, r, adminToken, rootDir, "kept.txt", "pw-1")
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "other.txt"), []byte("other"), 0644))

	post := func(path, body string) (int, string) {
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Authorization", adminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		_, code, msg := parseRespEnvelope(t, w)
		return code, msg
	}
	infoCode := func(id string) int {
		req := httptest.NewRequest("GET", "/api/public/share/info?share_id="+id, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		_, code, _ := parseRespEnvelope(t, w)
		return code
	}

	require.Equal(t, 200, infoCode(shareID))

	code, _ := post("/api/share/disable", fmt.Sprintf(`{"share_id":%q}`, shareID))
	require.Equal(t, 200, code)
	assert.Equal(t, 404, infoCode(shareID), "a disabled link must stop working")

	// While the link is off, its slug stays taken. Every attempt to reuse it, by
	// creating or by renaming, must fail with the same message, every time.
	var messages []string
	for i := 0; i < 3; i++ {
		code, msg := post("/api/share/create", fmt.Sprintf(`{"path":"/other.txt","share_id":%q}`, shareID))
		assert.Equal(t, 400, code, "attempt %d", i)
		messages = append(messages, msg)
	}
	otherID, _ := createShareFixture(t, r, adminToken, rootDir, "third.txt", "pw-3")
	code, msg := post("/api/share/update", fmt.Sprintf(`{"share_id":%q,"new_share_id":%q}`, otherID, shareID))
	assert.Equal(t, 400, code)
	messages = append(messages, msg)
	for _, m := range messages {
		assert.Equal(t, "share link already exists", m)
	}

	// Enabling brings back the very same link.
	code, _ = post("/api/share/enable", fmt.Sprintf(`{"share_id":%q}`, shareID))
	require.Equal(t, 200, code)
	assert.Equal(t, 200, infoCode(shareID), "the same slug must work again")

	// Enabling twice, or enabling a share that was never disabled, is harmless.
	code, _ = post("/api/share/enable", fmt.Sprintf(`{"share_id":%q}`, shareID))
	assert.Equal(t, 200, code)

	// A link that is on again keeps refusing the slug to others.
	code, msg = post("/api/share/create", fmt.Sprintf(`{"path":"/other.txt","share_id":%q}`, shareID))
	assert.Equal(t, 400, code)
	assert.Equal(t, "share link already exists", msg)

	// Deleting frees the slug for good.
	code, _ = post("/api/share/delete", fmt.Sprintf(`{"share_id":%q}`, shareID))
	require.Equal(t, 200, code)
	code, _ = post("/api/share/create", fmt.Sprintf(`{"path":"/other.txt","share_id":%q}`, shareID))
	assert.Equal(t, 200, code, "a deleted share's slug can be used again")
}

// A share with an access limit that ran out is switched off by the server (the
// record stays, unlike a burn-after-read share). Enabling it
// again must not leave it in a state where the next request switches it off
// once more, or serves it with a stale counter.
func TestEnablingAConsumedShareStartsAFreshBudget(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "once.txt"), []byte("one-time-payload"), 0644))

	post := func(path, body string) int {
		req := httptest.NewRequest("POST", path, strings.NewReader(body))
		req.Header.Set("Authorization", adminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		_, code, _ := parseRespEnvelope(t, w)
		return code
	}
	require.Equal(t, 200, post("/api/share/create", `{"path":"/once.txt","share_id":"limit-once","access_limit":1}`))

	download := func() (int, string) {
		req := httptest.NewRequest("GET", "/sd/limit-once", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code, w.Body.String()
	}
	status, body := download()
	require.Equal(t, 200, status)
	require.Equal(t, "one-time-payload", body)

	// Spent: refused with the "consumed" answer.
	_, body = download()
	assert.Contains(t, body, `"code":410`)

	// Enable: a fresh budget, and the link serves once more.
	require.Equal(t, 200, post("/api/share/enable", `{"share_id":"limit-once"}`))
	status, body = download()
	assert.Equal(t, 200, status)
	assert.Equal(t, "one-time-payload", body)

	// And it is burnt again afterwards, not left open.
	_, body = download()
	assert.Contains(t, body, `"code":410`)
}

func TestEnableUnknownShareIsNotFound(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)
	for _, body := range []string{`{"share_id":"does-not-exist"}`, `{"share_id":""}`, `{}`} {
		req := httptest.NewRequest("POST", "/api/share/enable", strings.NewReader(body))
		req.Header.Set("Authorization", adminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		require.NotPanics(t, func() { r.ServeHTTP(w, req) })
		_, code, _ := parseRespEnvelope(t, w)
		assert.Containsf(t, []int{400, 404}, code, "body %s", body)
	}
}
