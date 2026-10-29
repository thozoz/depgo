package handles_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Depgo only registers the API its own UI uses. The AList management surface
// (users, roles, storages, drivers, settings, offline download, search index,
// tasks, labels, SSO/LDAP/WebAuthn, archive extraction) either cannot work with a
// local disk and one administrator or answers with plausible data that has
// nothing to do with this deployment, so it must stay unregistered. An
// unregistered /api path answers 404; it never returns an API envelope.
func TestOnlyDepgoAPIRoutesAreRegistered(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)

	call := func(method, target string) (status int, contentType string) {
		req := httptest.NewRequest(method, target, strings.NewReader("{}"))
		req.Header.Set("Authorization", adminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code, w.Header().Get("Content-Type")
	}
	isJSON := func(ct string) bool { return strings.HasPrefix(ct, "application/json") }

	removed := [][2]string{
		{"GET", "/api/admin/user/list"},
		{"POST", "/api/admin/user/del_cache?username=admin"},
		{"GET", "/api/admin/role/list"},
		{"GET", "/api/admin/storage/list"},
		{"GET", "/api/admin/driver/list"},
		{"GET", "/api/admin/setting/list"},
		{"POST", "/api/admin/setting/reset_token"},
		{"GET", "/api/admin/meta/list"},
		{"POST", "/api/admin/index/build"},
		{"GET", "/api/admin/session/list"},
		{"GET", "/api/public/settings"},
		{"GET", "/api/public/offline_download_tools"},
		{"GET", "/api/public/archive_extensions"},
		{"POST", "/api/auth/login/ldap"},
		{"POST", "/api/auth/register"},
		{"GET", "/api/auth/sso"},
		{"GET", "/api/authn/webauthn_begin_login"},
		{"GET", "/api/me/sshkey/list"},
		{"POST", "/api/me/2fa/generate"},
		{"POST", "/api/fs/search"},
		{"POST", "/api/fs/other"},
		{"POST", "/api/fs/copy"},
		{"POST", "/api/fs/add_offline_download"},
		{"POST", "/api/fs/archive/decompress"},
		{"GET", "/api/task/upload/undone"},
		{"GET", "/api/label/list"},
		{"GET", "/api/label_file_binding/list"},
		{"GET", "/i/anything"},
	}
	for _, rt := range removed {
		status, ct := call(rt[0], rt[1])
		assert.Falsef(t, isJSON(ct), "%s %s must not be registered", rt[0], rt[1])
		if strings.HasPrefix(rt[1], "/api/") {
			assert.Equalf(t, 404, status, "%s %s must answer 404, not the SPA shell", rt[0], rt[1])
		}
	}

	kept := [][2]string{
		{"GET", "/api/me"},
		{"GET", "/api/me/sessions"},
		{"POST", "/api/fs/list"},
		{"POST", "/api/fs/mkdir"},
		{"GET", "/api/fs/trash"},
		{"GET", "/api/fs/versions"},
		{"GET", "/api/share/list"},
		{"GET", "/api/public/share/info?share_id=missing"},
	}
	for _, rt := range kept {
		_, ct := call(rt[0], rt[1])
		assert.Truef(t, isJSON(ct), "%s %s must stay registered", rt[0], rt[1])
	}
}
