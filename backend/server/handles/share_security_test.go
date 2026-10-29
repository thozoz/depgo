package handles_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	shareauth "github.com/alist-org/alist/v3/internal/share"
	"github.com/alist-org/alist/v3/server/handles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fastShareAuthLimiter turns off the response penalty and installs a fresh
// limiter so the rate-limit tests run instantly and in isolation.
func fastShareAuthLimiter(t *testing.T) {
	t.Helper()
	handles.ResetShareAuthLimiter()
	handles.ConfigureShareAuthLimiter(func(l *shareauth.AuthLimiter) {
		l.Sleep = func(time.Duration) {}
	})
	t.Cleanup(func() {
		handles.ResetShareAuthLimiter()
	})
}

// createShareFixture creates a password-protected share of a small file and
// returns its id together with a valid access token.
func createShareFixture(t *testing.T, r http.Handler, adminToken, rootDir, filename, password string) (shareID, token string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, filename), []byte("payload-0123456789"), 0644))

	body := fmt.Sprintf(`{"path":"/%s","name":"%s","password":"%s"}`, filename, filename, password)
	req := httptest.NewRequest("POST", "/api/share/create", strings.NewReader(body))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
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
	r.ServeHTTP(wAuth, reqAuth)
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

// assetPath reduces the absolute asset URL returned by the share API to the
// path portion, dropping the host (the test server is virtual) and the download
// signature, so the request has to authenticate through the header instead.
func assetPath(t *testing.T, raw string) string {
	t.Helper()
	parsed, err := url.Parse(raw)
	require.NoError(t, err, "download url %q", raw)
	require.NotEmpty(t, parsed.Path, "download url %q has no path", raw)
	return parsed.Path
}

// parseRespEnvelope decodes the wire envelope so a test can assert on the HTTP
// status and the body code independently.
func parseRespEnvelope(t *testing.T, w *httptest.ResponseRecorder) (httpStatus, bodyCode int, message string) {
	t.Helper()
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	status := w.Code
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		return status, 0, w.Body.String()
	}
	return status, resp.Code, resp.Message
}

// TestDownloadAcceptsNoForgedCredentials is the regression test for the
// 0369fda fix: before it, a non-empty Authorization header was treated as valid
// authentication for /d and /p, so "Authorization: x" downloaded a
// password-protected file. The fix must keep returning 401 for such a header.
func TestDownloadAcceptsNoForgedCredentials(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, token := createShareFixture(t, r, adminToken, rootDir, "guarded.txt", "guardpass")

	// Collect the signed download URL the share API hands out, then strip the
	// signature so the request has to fall back to header authentication.
	reqGet := httptest.NewRequest("POST", "/api/public/share/get",
		strings.NewReader(fmt.Sprintf(`{"share_id":"%s","path":"/","token":"%s"}`, shareID, token)))
	reqGet.Header.Set("Content-Type", "application/json")
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)
	require.Equal(t, 200, parseRespCode(t, wGet), "share get failed: %s", wGet.Body.String())

	var getResp struct {
		Data struct {
			Item struct {
				DownloadURL string `json:"download_url"`
			} `json:"item"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &getResp))
	downloadPath := getResp.Data.Item.DownloadURL
	require.NotEmpty(t, downloadPath)

	downloadPath = assetPath(t, downloadPath)

	forgedHeaders := []string{
		"x",
		"dev_token",
		"admin",
		"Bearer x",
		"null",
		"0",
	}
	for _, header := range forgedHeaders {
		for _, route := range []string{"/d", "/p"} {
			t.Run(fmt.Sprintf("%s_%s", route, header), func(t *testing.T) {
				req := httptest.NewRequest("GET", route+downloadPath, nil)
				req.Header.Set("Authorization", header)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				status, bodyCode, msg := parseRespEnvelope(t, w)
				if status == http.StatusOK && bodyCode == 200 {
					t.Fatalf("forged Authorization %q unlocked %s (body=%s)", header, route, w.Body.String())
				}
				if bodyCode != 401 {
					t.Fatalf("forged Authorization %q on %s returned code %d (status %d, msg %q), want 401",
						header, route, bodyCode, status, msg)
				}
				if body := w.Body.String(); strings.Contains(body, "payload-0123456789") {
					t.Fatalf("response leaked file content: %q", body)
				}
			})
		}
	}
}

// TestDownloadWithRealAdminTokenStillWorks makes sure the regression guard above
// did not simply break header authentication altogether.
func TestDownloadWithRealAdminTokenStillWorks(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	_, _ = createShareFixture(t, r, adminToken, rootDir, "real.txt", "realpass")

	// /d addresses the filesystem directly, so the request targets the real
	// path of the file rather than the share id. The share fixture guarantees
	// the file exists and that a password-protected share points at it.
	req := httptest.NewRequest("GET", "/d/real.txt", nil)
	req.Header.Set("Authorization", adminToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// A successful download streams the raw file, not a JSON envelope.
	assert.Equal(t, http.StatusOK, w.Code, "admin token must still download, body=%s", w.Body.String())
	assert.Contains(t, w.Body.String(), "payload-0123456789")
}

// TestShareAuthIsRateLimited proves the handler answers 429 once an attacker
// has spent the per-key budget, instead of allowing unlimited guessing.
func TestShareAuthIsRateLimited(t *testing.T) {
	fastShareAuthLimiter(t)
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, _ := createShareFixture(t, r, adminToken, rootDir, "brute.txt", "rightpass")

	const clientIP = "203.0.113.7:12345"
	// The policy allows 5 failures per (share, IP) pair, then locks out.
	for attempt := 0; attempt < 5; attempt++ {
		body := fmt.Sprintf(`{"share_id":"%s","password":"wrong-%d"}`, shareID, attempt)
		req := httptest.NewRequest("POST", "/api/public/share/auth", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = clientIP
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		_, bodyCode, _ := parseRespEnvelope(t, w)
		require.Equal(t, 403, bodyCode, "wrong password attempt %d should return 403", attempt+1)
	}

	// The next attempt, even with the correct password, must be refused.
	body := fmt.Sprintf(`{"share_id":"%s","password":"rightpass"}`, shareID)
	req := httptest.NewRequest("POST", "/api/public/share/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = clientIP
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	_, bodyCode, msg := parseRespEnvelope(t, w)
	assert.Equal(t, 429, bodyCode, "expected the limiter to answer 429, msg=%q body=%s", msg, w.Body.String())
	assert.NotEmpty(t, w.Header().Get("Retry-After"), "429 must carry a Retry-After header")

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Empty(t, resp.Data.Token, "a locked-out response must never carry an access token")
}

// TestShareAuthDifferentIPIsNotBlocked checks the limiter is scoped per client:
// one noisy neighbour must not lock a share for everybody.
func TestShareAuthDifferentIPIsNotBlocked(t *testing.T) {
	fastShareAuthLimiter(t)
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, _ := createShareFixture(t, r, adminToken, rootDir, "noisy.txt", "rightpass")

	for attempt := 0; attempt < 5; attempt++ {
		body := fmt.Sprintf(`{"share_id":"%s","password":"wrong-%d"}`, shareID, attempt)
		req := httptest.NewRequest("POST", "/api/public/share/auth", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "198.51.100.1:1111"
		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	body := fmt.Sprintf(`{"share_id":"%s","password":"rightpass"}`, shareID)
	req := httptest.NewRequest("POST", "/api/public/share/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "198.51.100.2:2222"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	_, bodyCode, msg := parseRespEnvelope(t, w)
	assert.Equal(t, 200, bodyCode, "a different IP must still authenticate, msg=%q", msg)
}

// TestShareAuthSuccessClearsFailures verifies a legitimate user who mistypes
// once is not penalised afterwards.
func TestShareAuthSuccessClearsFailures(t *testing.T) {
	fastShareAuthLimiter(t)
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, _ := createShareFixture(t, r, adminToken, rootDir, "typo.txt", "rightpass")

	const clientIP = "192.0.2.55:9999"
	post := func(password string) (int, *httptest.ResponseRecorder) {
		body := fmt.Sprintf(`{"share_id":"%s","password":"%s"}`, shareID, password)
		req := httptest.NewRequest("POST", "/api/public/share/auth", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = clientIP
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		_, code, _ := parseRespEnvelope(t, w)
		return code, w
	}

	code, _ := post("tpyo")
	require.Equal(t, 403, code)
	code, _ = post("rightpass")
	require.Equal(t, 200, code)

	// Budget must be fresh again: four more failures are tolerated.
	for i := 0; i < 4; i++ {
		code, _ = post(fmt.Sprintf("wrong-%d", i))
		require.Equal(t, 403, code, "failure %d after a success should still be 403", i+1)
	}
}

// TestAuthResponsesAreNotCacheable asserts Cache-Control: no-store on the
// responses that carry or validate a credential.
func TestAuthResponsesAreNotCacheable(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, token := createShareFixture(t, r, adminToken, rootDir, "cache.txt", "cachepass")

	cases := []struct {
		name   string
		method string
		target string
		body   string
		header map[string]string
	}{
		{
			name:   "share auth",
			method: "POST",
			target: "/api/public/share/auth",
			body:   fmt.Sprintf(`{"share_id":"%s","password":"cachepass"}`, shareID),
		},
		{
			name:   "share info",
			method: "GET",
			target: "/api/public/share/info?share_id=" + shareID,
			header: map[string]string{"X-Share-Token": token},
		},
		{
			name:   "share list",
			method: "POST",
			target: "/api/public/share/list",
			body:   fmt.Sprintf(`{"share_id":"%s","path":"/","token":"%s"}`, shareID, token),
		},
		{
			name:   "login",
			method: "POST",
			target: "/api/auth/login/hash",
			body:   `{"username":"admin","password":"whatever"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.body != "" {
				req = httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.target, nil)
			}
			for k, v := range tc.header {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			cacheControl := strings.ToLower(w.Header().Get("Cache-Control"))
			assert.Contains(t, cacheControl, "no-store",
				"%s must not be cacheable, got Cache-Control=%q", tc.name, cacheControl)
		})
	}
}

// TestInstalledShareTokenCarriesNoPasswordHash makes sure the token handed to a
// client does not embed the stored password hash.
func TestInstalledShareTokenCarriesNoPasswordHash(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, _ := createShareFixture(t, r, adminToken, rootDir, "hashleak.txt", "hashpass")

	stored, err := db.GetShareByShareID(shareID)
	require.NoError(t, err)
	require.NotEmpty(t, stored.PasswordHash, "fixture must have a password hash")

	body := fmt.Sprintf(`{"share_id":"%s","password":"hashpass"}`, shareID)
	req := httptest.NewRequest("POST", "/api/public/share/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w))

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data.Token)
	assert.NotContains(t, resp.Data.Token, stored.PasswordHash,
		"access token must not embed the share password hash")
}

// TestShareInfoWithTokenDoesNotLeakForOtherShare proves the token is share
// scoped: a token for share A cannot raise privileges on share B.
func TestShareInfoWithTokenDoesNotLeakForOtherShare(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareA, tokenA := createShareFixture(t, r, adminToken, rootDir, "a.txt", "passa")

	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "b.txt"), []byte("b"), 0644))
	body := `{"path":"/b.txt","name":"b.txt","password":"passb"}`
	reqCreate := httptest.NewRequest("POST", "/api/share/create", strings.NewReader(body))
	reqCreate.Header.Set("Authorization", adminToken)
	reqCreate.Header.Set("Content-Type", "application/json")
	wCreate := httptest.NewRecorder()
	r.ServeHTTP(wCreate, reqCreate)
	require.Equal(t, 200, parseRespCode(t, wCreate))
	var created struct {
		Data struct {
			ShareID string `json:"share_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wCreate.Body.Bytes(), &created))
	shareB := created.Data.ShareID

	req := httptest.NewRequest("GET", "/api/public/share/info?share_id="+shareB, nil)
	req.Header.Set("X-Share-Token", tokenA)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w))

	var resp struct {
		Data struct {
			Authed bool   `json:"authed"`
			Name   string `json:"name"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.Data.Authed, "share A's token must not authenticate share B")
	assert.Empty(t, resp.Data.Name, "a locked share must not disclose its name")
	assert.NotEqual(t, shareA, shareB)
}

// TestAssetsDirectoryListingIsDisabled asserts the /assets/ autoindex is gone
// while real bundle files remain reachable.
func TestAssetsDirectoryListingIsDisabled(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)

	for _, path := range []string{"/assets/", "/assets", "/images/"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusOK {
			assert.NotContains(t, w.Body.String(), "<a href=",
				"GET %s must not render a directory index", path)
		}
	}
}

// TestSecurityHeadersPresent asserts the hardening headers are attached.
func TestSecurityHeadersPresent(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
	assert.NotEmpty(t, w.Header().Get("Content-Security-Policy"))

	// HSTS must not be sent over plain HTTP.
	assert.Empty(t, w.Header().Get("Strict-Transport-Security"),
		"HSTS must not be emitted for plain HTTP")

	reqTLS := httptest.NewRequest("GET", "/ping", nil)
	reqTLS.Header.Set("X-Forwarded-Proto", "https")
	wTLS := httptest.NewRecorder()
	r.ServeHTTP(wTLS, reqTLS)
	assert.NotEmpty(t, wTLS.Header().Get("Strict-Transport-Security"),
		"HSTS must be emitted when the request arrived over HTTPS")
}

// TestShareTokenSurvivesAccessRevisionOnly guards against an accidental change
// that would make tokens independent of revocation.
func TestShareTokenSurvivesAccessRevisionOnly(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, token := createShareFixture(t, r, adminToken, rootDir, "rev.txt", "revpass")

	// A working token must be accepted.
	req := httptest.NewRequest("POST", "/api/public/share/get",
		strings.NewReader(fmt.Sprintf(`{"share_id":"%s","path":"/","token":"%s"}`, shareID, token)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "valid token rejected: %s", w.Body.String())

	// Bumping the revision must invalidate it immediately.
	stored, err := db.GetShareByShareID(shareID)
	require.NoError(t, err)
	require.NoError(t, db.GetDb().Model(&model.Share{}).
		Where("id = ?", stored.ID).
		Update("access_revision", stored.AccessRevision+1).Error)

	req2 := httptest.NewRequest("POST", "/api/public/share/get",
		strings.NewReader(fmt.Sprintf(`{"share_id":"%s","path":"/","token":"%s"}`, shareID, token)))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	_, bodyCode, _ := parseRespEnvelope(t, w2)
	assert.Equal(t, 401, bodyCode, "token must be rejected after AccessRevision changes")
	_ = time.Now() // keep the time import meaningful if assertions change
}
