package handles_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/server/handles"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// adminUserUpdateRoute wires the administrative update handler directly.
//
// Depgo deliberately does not register the AList management API (see
// routes_surface_test.go), so this exercises the handler the way the upstream
// route would, without re-enabling an endpoint this deployment does not serve.
func adminUserUpdateRoute() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/admin/user/update", handles.UpdateUser)
	return r
}

func adminUpdate(t *testing.T, r http.Handler, body string) (int, int, string) {
	t.Helper()
	req := httptest.NewRequest("POST", "/admin/user/update", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return parseRespEnvelope(t, w)
}

// TestPartialUserUpdateKeepsUnsentFields is the regression test for a partial
// update that bound the body straight into model.User: fields the client did not
// send arrived as zero values and were saved, so resetting a password stripped
// the account's roles, permission mask and 2FA secret.
func TestPartialUserUpdateKeepsUnsentFields(t *testing.T) {
	setupTestServer(t)
	seedTestRole(t)

	user := &model.User{
		ID: 600, Username: "partial", Role: model.Roles{testRoleID},
		Permission: 0x00FF, BasePath: "/", Authn: "[]", OtpSecret: "JBSWY3DPEHPK3PXP",
	}
	user.SetPassword("old-password")
	require.NoError(t, db.CreateUser(user))

	r := adminUserUpdateRoute()
	_, code, msg := adminUpdate(t, r, `{"id":600,"password":"new-password"}`)
	require.Equal(t, 200, code, "password reset failed: %s", msg)

	after, err := db.GetUserById(600)
	require.NoError(t, err)
	assert.Equal(t, model.Roles{testRoleID}, after.Role, "roles must survive a partial update")
	assert.Equal(t, int32(0x00FF), after.Permission, "permission must survive a partial update")
	assert.Equal(t, "JBSWY3DPEHPK3PXP", after.OtpSecret, "an absent otp_secret must not clear 2FA")
	assert.False(t, after.Disabled)
	assert.NoError(t, after.ValidateRawPassword("new-password"))
}

// TestPartialUserUpdateCanChangeFields proves the pointer fields are still
// settable when they are actually sent.
func TestPartialUserUpdateCanChangeFields(t *testing.T) {
	setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 601, Username: "fields", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]", OtpSecret: "OLD"}
	user.SetPassword("fields-password")
	require.NoError(t, db.CreateUser(user))

	r := adminUserUpdateRoute()
	body := `{"id":601,"username":"fields-renamed","permission":5,"otp_secret":"NEWSECRET","disabled":true}`
	_, code, msg := adminUpdate(t, r, body)
	require.Equal(t, 200, code, "update failed: %s", msg)

	after, err := db.GetUserById(601)
	require.NoError(t, err)
	assert.Equal(t, "fields-renamed", after.Username)
	assert.Equal(t, int32(5), after.Permission)
	assert.Equal(t, "NEWSECRET", after.OtpSecret)
	assert.True(t, after.Disabled)
}

// TestUserUpdateRejectsPrivilegeEscalation keeps the management endpoint from
// minting another administrator (or a guest) through a role assignment.
func TestUserUpdateRejectsPrivilegeEscalation(t *testing.T) {
	setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 602, Username: "escalate", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]"}
	user.SetPassword("escalate-password")
	require.NoError(t, db.CreateUser(user))

	r := adminUserUpdateRoute()
	for _, role := range []string{
		fmt.Sprintf(`[%d]`, model.ADMIN),
		fmt.Sprintf(`[%d]`, model.GUEST),
	} {
		_, code, _ := adminUpdate(t, r, fmt.Sprintf(`{"id":602,"role":%s}`, role))
		assert.NotEqual(t, 200, code, "role %s must not be assignable", role)

		after, err := db.GetUserById(602)
		require.NoError(t, err)
		assert.False(t, after.IsAdmin(), "the user must not have become an admin")
	}
}

// TestAdminRoleCannotBeChanged proves the admin-role guard now compares the
// stored role instead of the zero value from an absent field.
func TestAdminRoleCannotBeChanged(t *testing.T) {
	setupTestServer(t)
	seedTestRole(t)

	// setupTestServer seeds the administrator with ID 1.
	r := adminUserUpdateRoute()

	// An update that does not touch the role must still be allowed.
	_, code, msg := adminUpdate(t, r, `{"id":1,"username":"admin"}`)
	require.Equal(t, 200, code, "a no-op admin update must be allowed: %s", msg)

	// A role change on the admin account must be refused.
	_, code, _ = adminUpdate(t, r, fmt.Sprintf(`{"id":1,"role":[%d]}`, testRoleID))
	assert.NotEqual(t, 200, code, "the admin role must not be changeable")

	after, err := db.GetUserById(1)
	require.NoError(t, err)
	assert.True(t, after.IsAdmin(), "the admin must keep its role")
}

// TestUserUpdateDoesNotClearOtpSecret covers the explicit 2FA downgrade: an empty
// otp_secret means "leave it alone" rather than "disable 2FA".
func TestUserUpdateDoesNotClearOtpSecret(t *testing.T) {
	setupTestServer(t)
	seedTestRole(t)

	user := &model.User{
		ID: 603, Username: "otpuser", Role: model.Roles{testRoleID},
		BasePath: "/", Authn: "[]", OtpSecret: "KEEPME",
	}
	user.SetPassword("otp-password")
	require.NoError(t, db.CreateUser(user))

	r := adminUserUpdateRoute()
	_, code, msg := adminUpdate(t, r, `{"id":603,"otp_secret":""}`)
	require.Equal(t, 200, code, "update failed: %s", msg)

	after, err := db.GetUserById(603)
	require.NoError(t, err)
	assert.Equal(t, "KEEPME", after.OtpSecret, "an empty otp_secret must not disable 2FA")
}
