package handles_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedLimiterRole creates a role scoped to one path and clears the role cache.
func seedLimiterRole(t *testing.T, id uint, name, scope string, permission int32) {
	t.Helper()
	if _, err := db.GetRole(id); err != nil {
		require.NoError(t, db.CreateRole(&model.Role{
			ID:               id,
			Name:             name,
			PermissionScopes: []model.PermissionEntry{{Path: scope, Permission: permission}},
		}))
	}
	op.ClearRoleCache()
	t.Cleanup(op.ClearRoleCache)
}

// TestDownloadRejectsPathOutsideBasePath is the integration regression test for a
// download IDOR: /d and /p take the URL path verbatim, so without an explicit
// permission check any signed-in user could fetch any file by naming it.
func TestDownloadRejectsPathOutsideBasePath(t *testing.T) {
	r, _, _, rootDir, _ := setupTestServer(t)
	seedLimiterRole(t, 150, "public-reader", "/public", 0x1)

	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "private"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "private", "secret.txt"), []byte("TOP-SECRET-DATA"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "public"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "public", "open.txt"), []byte("PUBLIC-DATA"), 0644))

	user := &model.User{
		ID: 500, Username: "limited", Role: model.Roles{150},
		Permission: 0x1, BasePath: "/public", Authn: "[]",
	}
	user.SetPassword("limited-password")
	require.NoError(t, db.CreateUser(user))

	token := loginUser(t, r, "limited", "limited-password")

	// The forbidden path must not be served.
	for _, route := range []string{"/d", "/p"} {
		t.Run("forbidden"+route, func(t *testing.T) {
			req := httptest.NewRequest("GET", route+"/private/secret.txt", nil)
			req.Header.Set("Authorization", token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			body := w.Body.String()
			assert.NotContains(t, body, "TOP-SECRET-DATA",
				"%s must not serve a file outside the base path, body=%q", route, body)
		})
	}

	// The path the user does own must still work.
	t.Run("allowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/d/public/open.txt", nil)
		req.Header.Set("Authorization", token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "body=%q", w.Body.String())
		assert.Contains(t, w.Body.String(), "PUBLIC-DATA")
	})
}

// TestDownloadAdminTokenStillUnrestricted confirms the fix did not remove the
// admin token's ability to download any path.
func TestDownloadAdminTokenStillUnrestricted(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)

	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "private"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "private", "secret.txt"), []byte("TOP-SECRET-DATA"), 0644))

	req := httptest.NewRequest("GET", "/d/private/secret.txt", nil)
	req.Header.Set("Authorization", adminToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "body=%q", w.Body.String())
	assert.Contains(t, w.Body.String(), "TOP-SECRET-DATA")
}

// TestSignedDownloadURLStillWorks proves the signature branch is untouched: a URL
// minted for a user continues to work without a bearer token.
func TestSignedDownloadURLStillWorks(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)

	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "private"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "private", "secret.txt"), []byte("TOP-SECRET-DATA"), 0644))

	// Force every link to carry a signature, so fs/get returns a signed URL.
	require.NoError(t, op.SaveSettingItem(&model.SettingItem{Key: conf.SignAll, Value: "true", Type: conf.TypeBool, Group: model.GLOBAL, Flag: model.PRIVATE}))
	op.SettingCacheUpdate()

	// The admin file API mints a signed URL after performing permission checks.
	body := `{"path":"/private/secret.txt"}`
	reqGet := httptest.NewRequest("POST", "/api/fs/get", strings.NewReader(body))
	reqGet.Header.Set("Authorization", adminToken)
	reqGet.Header.Set("Content-Type", "application/json")
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)
	require.Equal(t, 200, parseRespCode(t, wGet), "fs/get failed: %s", wGet.Body.String())

	var resp struct {
		Data struct {
			RawURL string `json:"raw_url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data.RawURL, "expected a signed url: %s", wGet.Body.String())

	path := resp.Data.RawURL
	if i := strings.Index(path, "://"); i >= 0 {
		rest := path[i+3:]
		if j := strings.Index(rest, "/"); j >= 0 {
			path = rest[j:]
		}
	}

	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "signed url %q must work, body=%q", path, w.Body.String())
	assert.Contains(t, w.Body.String(), "TOP-SECRET-DATA")
}
