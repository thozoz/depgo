package handles_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRoleID is a self-contained role for these tests. Built-in ids such as
// model.GENERAL are enum constants, not guaranteed database rows: the bootstrap
// seeds guest/admin with their own numeric ids, so a test user carrying
// model.GENERAL would fail role resolution in the auth middleware.
const testRoleID = 130

// seedTestRole creates the role used by the users below. It must run after
// setupTestServer, which is what initialises the database.
func seedTestRole(t *testing.T) {
	t.Helper()
	if _, err := db.GetRole(testRoleID); err != nil {
		require.NoError(t, db.CreateRole(&model.Role{
			ID:               testRoleID,
			Name:             "test-readers",
			Description:      "role used by handler tests",
			PermissionScopes: []model.PermissionEntry{{Path: "/", Permission: 0x1}},
		}))
	}
	op.ClearRoleCache()
	t.Cleanup(op.ClearRoleCache)
}

// loginUser signs in and returns the session token.
func loginUser(t *testing.T, handler http.Handler, username, password string) string {
	t.Helper()
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "login failed: %s", w.Body.String())

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data.Token)
	return resp.Data.Token
}

// TestProfileUpdateCannotRebindSSO is the regression test for an account
// takeover: the profile endpoint accepted sso_id, while SSO login resolves the
// account from that very column. A user could therefore write another user's
// SSO identifier into their own row and, on their next SSO sign-in, be handed
// that user's session.
func TestProfileUpdateCannotRebindSSO(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	const victimSSO = "sso-victim-4711"
	victim := &model.User{
		ID: 80, Username: "victimuser", Role: model.Roles{testRoleID},
		Permission: 0x1, BasePath: "/", Authn: "[]", SsoID: victimSSO,
	}
	victim.SetPassword("victim-password")
	require.NoError(t, db.CreateUser(victim))

	attacker := &model.User{
		ID: 81, Username: "attackeruser", Role: model.Roles{testRoleID},
		Permission: 0x1, BasePath: "/", Authn: "[]",
	}
	attacker.SetPassword("attacker-password")
	require.NoError(t, db.CreateUser(attacker))

	token := loginUser(t, r, "attackeruser", "attacker-password")

	body := fmt.Sprintf(`{"username":"attackeruser","sso_id":%q}`, victimSSO)
	req := httptest.NewRequest("POST", "/api/me/update", strings.NewReader(body))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	after, err := db.GetUserById(81)
	require.NoError(t, err)
	assert.Empty(t, after.SsoID, "a profile update must not be able to bind an SSO identity")

	resolved, err := db.GetUserBySSOID(victimSSO)
	require.NoError(t, err)
	assert.Equal(t, victim.ID, resolved.ID, "SSO id must keep resolving to the original owner")
}

// TestProfileUpdateKeepsRolesAndPermission proves the self-service update only
// touches username and password.
func TestProfileUpdateKeepsRolesAndPermission(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{
		ID: 82, Username: "profileuser", Role: model.Roles{testRoleID},
		Permission: 0x00AA, BasePath: "/", Authn: "[]", SsoID: "keep-me",
	}
	user.SetPassword("profile-password")
	require.NoError(t, db.CreateUser(user))

	token := loginUser(t, r, "profileuser", "profile-password")

	body := `{"username":"profileuser2","password":"new-profile-password","current_password":"profile-password"}`
	req := httptest.NewRequest("POST", "/api/me/update", strings.NewReader(body))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "profile update failed: %s", w.Body.String())

	after, err := db.GetUserById(82)
	require.NoError(t, err)
	assert.Equal(t, "profileuser2", after.Username)
	assert.Equal(t, model.Roles{testRoleID}, after.Role, "roles must survive a profile update")
	assert.Equal(t, int32(0x00AA), after.Permission, "permission must survive a profile update")
	assert.Equal(t, "keep-me", after.SsoID, "an existing SSO binding must survive a profile update")
	assert.NoError(t, after.ValidateRawPassword("new-profile-password"))
}

// TestProfileUpdateRejectsDuplicateUsername keeps two accounts from ending up
// with the same name, which would make login and token validation ambiguous.
func TestProfileUpdateRejectsDuplicateUsername(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	taken := &model.User{ID: 83, Username: "takenname", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]"}
	taken.SetPassword("taken-password")
	require.NoError(t, db.CreateUser(taken))

	me := &model.User{ID: 84, Username: "renameme", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]"}
	me.SetPassword("rename-password")
	require.NoError(t, db.CreateUser(me))

	token := loginUser(t, r, "renameme", "rename-password")
	body := `{"username":"takenname"}`
	req := httptest.NewRequest("POST", "/api/me/update", strings.NewReader(body))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	_, bodyCode, msg := parseRespEnvelope(t, w)
	assert.NotEqual(t, 200, bodyCode, "renaming onto an existing username must be refused, msg=%q", msg)

	after, err := db.GetUserById(84)
	require.NoError(t, err)
	assert.Equal(t, "renameme", after.Username)
}

// TestProfileUpdateChangesPassword checks the happy path: a user can still
// rotate their own password, and the old one stops working.
func TestProfileUpdateChangesPassword(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 85, Username: "pwuser", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]"}
	user.SetPassword("old-password")
	require.NoError(t, db.CreateUser(user))

	token := loginUser(t, r, "pwuser", "old-password")
	body := `{"username":"pwuser","password":"a-much-better-password","current_password":"old-password"}`
	req := httptest.NewRequest("POST", "/api/me/update", strings.NewReader(body))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "password change failed: %s", w.Body.String())

	after, err := db.GetUserById(85)
	require.NoError(t, err)
	assert.NoError(t, after.ValidateRawPassword("a-much-better-password"))
	assert.Error(t, after.ValidateRawPassword("old-password"))

	// The new password must actually be usable.
	assert.NotEmpty(t, loginUser(t, r, "pwuser", "a-much-better-password"))
}

// TestProfileUpdatePasswordOnly is the regression test for a password change that
// required resending the username. A client that only wants to rotate its
// password must be able to do so.
func TestProfileUpdatePasswordOnly(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 86, Username: "pwdonly", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]"}
	user.SetPassword("original-password")
	require.NoError(t, db.CreateUser(user))

	token := loginUser(t, r, "pwdonly", "original-password")

	body := `{"password":"replacement-password","current_password":"original-password"}`
	req := httptest.NewRequest("POST", "/api/me/update", strings.NewReader(body))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "password-only update failed: %s", w.Body.String())

	after, err := db.GetUserById(86)
	require.NoError(t, err)
	assert.Equal(t, "pwdonly", after.Username, "username must be untouched")
	assert.NoError(t, after.ValidateRawPassword("replacement-password"))
	assert.NotEmpty(t, loginUser(t, r, "pwdonly", "replacement-password"),
		"the new password must work end to end")
}

// TestProfileUpdateUsernameOnly checks the mirror case: rename without touching
// the password.
func TestProfileUpdateUsernameOnly(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 87, Username: "renameonly", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]"}
	user.SetPassword("stable-password")
	require.NoError(t, db.CreateUser(user))

	token := loginUser(t, r, "renameonly", "stable-password")
	body := `{"username":"renamedonly"}`
	req := httptest.NewRequest("POST", "/api/me/update", strings.NewReader(body))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "rename failed: %s", w.Body.String())

	after, err := db.GetUserById(87)
	require.NoError(t, err)
	assert.Equal(t, "renamedonly", after.Username)
	assert.NoError(t, after.ValidateRawPassword("stable-password"), "the password must be unchanged")
}

// TestProfileUpdateRejectsEmptyInputs keeps an empty body or empty fields from
// being treated as a valid no-op update.
func TestProfileUpdateRejectsEmptyInputs(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 88, Username: "emptyfields", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]"}
	user.SetPassword("some-password")
	require.NoError(t, db.CreateUser(user))

	token := loginUser(t, r, "emptyfields", "some-password")

	for name, body := range map[string]string{
		"empty body":     `{}`,
		"empty password": `{"password":""}`,
		"empty username": `{"username":""}`,
		"blank username": `{"username":"   "}`,
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/me/update", strings.NewReader(body))
			req.Header.Set("Authorization", token)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			_, bodyCode, msg := parseRespEnvelope(t, w)
			assert.Equal(t, 400, bodyCode, "%s should be refused, msg=%q", name, msg)
		})
	}

	after, err := db.GetUserById(88)
	require.NoError(t, err)
	assert.Equal(t, "emptyfields", after.Username)
	assert.NoError(t, after.ValidateRawPassword("some-password"), "the password must be untouched")
}

// TestProfileUpdateRequiresCurrentPassword is the regression test for a stolen
// session token being enough to take ownership of an account: without proof of the
// current password, whoever holds the token can set a new one and lock the real
// owner out.
func TestProfileUpdateRequiresCurrentPassword(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 89, Username: "stolentoken", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]"}
	user.SetPassword("the-real-password")
	require.NoError(t, db.CreateUser(user))

	// An attacker who only has the session token.
	token := loginUser(t, r, "stolentoken", "the-real-password")

	cases := map[string]string{
		"no current password":    `{"password":"attacker-password"}`,
		"empty current password": `{"password":"attacker-password","current_password":""}`,
		"wrong current password": `{"password":"attacker-password","current_password":"guess"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/me/update", strings.NewReader(body))
			req.Header.Set("Authorization", token)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			_, bodyCode, msg := parseRespEnvelope(t, w)
			assert.NotEqual(t, 200, bodyCode, "%s must be refused, msg=%q", name, msg)
		})
	}

	after, err := db.GetUserById(89)
	require.NoError(t, err)
	assert.NoError(t, after.ValidateRawPassword("the-real-password"), "the original password must still work")
	assert.Error(t, after.ValidateRawPassword("attacker-password"), "the attacker's password must not be accepted")
}

// TestSiteAdminTokenCanSetPasswordWithoutCurrent documents the deliberate
// exemption: the site admin token already represents full control and has its own
// administrative reset path, so it is not asked for the old password.
func TestSiteAdminTokenCanSetPasswordWithoutCurrent(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 90, Username: "adminset", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]"}
	user.SetPassword("forgotten")
	require.NoError(t, db.CreateUser(user))

	body := `{"password":"set-by-admin-token"}`
	req := httptest.NewRequest("POST", "/api/me/update", strings.NewReader(body))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "admin token must be able to set a password: %s", w.Body.String())
}
