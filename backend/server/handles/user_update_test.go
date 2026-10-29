package handles_test

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedUser creates a non-admin user with a distinct role and permission mask so
// a test can detect fields being silently reset.
func seedUser(t *testing.T, id uint, username string, role model.Roles, permission int32, disabled bool) *model.User {
	t.Helper()
	u := &model.User{
		ID:         id,
		Username:   username,
		Role:       role,
		Permission: permission,
		Disabled:   disabled,
		BasePath:   "/",
		Authn:      "[]",
	}
	u.SetPassword("initial-password")
	require.NoError(t, db.CreateUser(u))
	return u
}

// TestPartialUserUpdateKeepsUnsentFields is the regression test for a partial
// update wiping Role and Permission. Before the fix, binding the request body
// straight into model.User meant an omitted field arrived as its zero value and
// was then persisted.
func TestPartialUserUpdateKeepsUnsentFields(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)

	const roleID = 101
	seedUser(t, 50, "bob", model.Roles{model.GENERAL, roleID}, 0x0FFF, false)

	// Only id and a new password: role, permission and disabled are absent.
	body := `{"id":50,"password":"brand-new-password"}`
	req := httptest.NewRequest("POST", "/api/admin/user/update", strings.NewReader(body))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "update failed: %s", w.Body.String())

	after, err := db.GetUserById(50)
	require.NoError(t, err)

	assert.Equal(t, "bob", after.Username, "username must be preserved when not sent")
	assert.Equal(t, model.Roles{model.GENERAL, roleID}, after.Role, "role must be preserved when not sent")
	assert.Equal(t, int32(0x0FFF), after.Permission, "permission must be preserved when not sent")
	assert.False(t, after.Disabled, "disabled must be preserved when not sent")
	assert.Equal(t, "[]", after.Authn, "webauthn credentials must be preserved")

	// The new password must actually work and the old one must not.
	assert.NoError(t, after.ValidateRawPassword("brand-new-password"))
	assert.Error(t, after.ValidateRawPassword("initial-password"))
}

// TestPartialUserUpdateCanChangeFields checks the fields that are sent do get
// applied, so the merge above is not simply ignoring the request.
func TestPartialUserUpdateCanChangeFields(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)

	otherRole := model.Role{ID: 120, Name: "readers", PermissionScopes: []model.PermissionEntry{{Path: "/", Permission: 0x1}}}
	require.NoError(t, db.CreateRole(&otherRole))
	seedUser(t, 51, "carol", model.Roles{model.GENERAL, 101}, 0x00FF, false)

	body := fmt.Sprintf(`{"id":51,"username":"carol-renamed","role":[0,%d],"permission":3,"disabled":true}`, otherRole.ID)
	req := httptest.NewRequest("POST", "/api/admin/user/update", strings.NewReader(body))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "update failed: %s", w.Body.String())

	after, err := db.GetUserById(51)
	require.NoError(t, err)
	assert.Equal(t, "carol-renamed", after.Username)
	assert.Equal(t, model.Roles{model.GENERAL, int(otherRole.ID)}, after.Role)
	assert.Equal(t, int32(3), after.Permission)
	assert.True(t, after.Disabled)
}

// TestUserUpdateRejectsPrivilegeEscalation verifies a caller cannot promote a
// user to admin through the update endpoint.
func TestUserUpdateRejectsPrivilegeEscalation(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)
	seedUser(t, 52, "dave", model.Roles{model.GENERAL, 101}, 0x00FF, false)

	body := `{"id":52,"role":[2]}`
	req := httptest.NewRequest("POST", "/api/admin/user/update", strings.NewReader(body))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	_, bodyCode, msg := parseRespEnvelope(t, w)
	assert.NotEqual(t, 200, bodyCode, "assigning the admin role must be refused, msg=%q", msg)

	after, err := db.GetUserById(52)
	require.NoError(t, err)
	assert.Equal(t, model.Roles{model.GENERAL, 101}, after.Role, "role must be unchanged after a refused promotion")
}

// TestAdminRoleCannotBeChanged verifies the built-in admin account keeps its
// roles, which would otherwise let an operator lock themselves out.
func TestAdminRoleCannotBeChanged(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)

	admin, err := db.GetUserById(1)
	require.NoError(t, err)
	before := append(model.Roles(nil), admin.Role...)

	body := `{"id":1,"role":[0]}`
	req := httptest.NewRequest("POST", "/api/admin/user/update", strings.NewReader(body))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	_, bodyCode, msg := parseRespEnvelope(t, w)
	assert.NotEqual(t, 200, bodyCode, "changing the admin role must be refused, msg=%q", msg)

	after, err := db.GetUserById(1)
	require.NoError(t, err)
	assert.Equal(t, before, after.Role)
}

// TestUserUpdateDoesNotClearOtpSecret makes sure an update without an
// otp_secret keeps two-factor authentication enabled.
func TestUserUpdateDoesNotClearOtpSecret(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)

	u := seedUser(t, 53, "erin", model.Roles{model.GENERAL, 101}, 0x00FF, false)
	u.OtpSecret = "JBSWY3DPEHPK3PXP"
	require.NoError(t, db.UpdateUser(u))

	body := `{"id":53,"permission":7}`
	req := httptest.NewRequest("POST", "/api/admin/user/update", strings.NewReader(body))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "update failed: %s", w.Body.String())

	after, err := db.GetUserById(53)
	require.NoError(t, err)
	assert.Equal(t, "JBSWY3DPEHPK3PXP", after.OtpSecret, "otp secret must survive an unrelated update")
	assert.Equal(t, int32(7), after.Permission)
}

// TestAdminCanResetUserPassword verifies the administrative password reset:
// an admin can set a new password for another user without touching any other
// field, and the user can then log in with it.
func TestAdminCanResetUserPassword(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{
		ID: 210, Username: "resetme", Role: model.Roles{testRoleID},
		Permission: 0x1, BasePath: "/", Authn: "[]",
	}
	user.SetPassword("forgotten-password")
	require.NoError(t, db.CreateUser(user))

	body := `{"id":210,"password":"admin-set-password"}`
	req := httptest.NewRequest("POST", "/api/admin/user/update", strings.NewReader(body))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "admin password reset failed: %s", w.Body.String())

	after, err := db.GetUserById(210)
	require.NoError(t, err)
	assert.NoError(t, after.ValidateRawPassword("admin-set-password"))
	assert.Error(t, after.ValidateRawPassword("forgotten-password"))
	// Nothing else may have moved.
	assert.Equal(t, "resetme", after.Username)
	assert.Equal(t, model.Roles{testRoleID}, after.Role)
	assert.Equal(t, int32(0x1), after.Permission)

	// The new password must work end to end, not just in the database.
	assert.NotEmpty(t, loginUser(t, r, "resetme", "admin-set-password"))
}
