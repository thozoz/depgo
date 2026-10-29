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

// `depgo admin set` runs in a separate process, so it can only change the
// database. The running server used to keep the administrator in memory (and
// every other user for an hour), so the old password kept working and the new one
// did not until the server restarted. This test changes the password straight in
// the database, the way the CLI does, and expects the very next request to see it.
func TestPasswordChangedOutsideTheServerTakesEffectImmediately(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 700, Username: "resetme", Role: model.Roles{testRoleID}, Permission: 0x1, BasePath: "/", Authn: "[]"}
	user.SetPassword("old-password")
	// The account existed before the reset; the password timestamp has one-second
	// resolution, so without this the two writes would share a timestamp.
	user.PwdTS -= 60
	require.NoError(t, db.CreateUser(user))

	oldToken := loginUser(t, r, "resetme", "old-password")

	// Warm whatever the server might cache about this user.
	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", oldToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), w.Body.String())

	// The "other process": load the row, set a new password, write it back.
	stored, err := db.GetUserByName("resetme")
	require.NoError(t, err)
	stored.SetPassword("new-password")
	require.NoError(t, db.UpdateUser(stored))

	login := func(password string) int {
		body := fmt.Sprintf(`{"username":"resetme","password":%q}`, password)
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return parseRespCode(t, w)
	}
	assert.NotEqual(t, 200, login("old-password"), "the old password must stop working at once")
	assert.Equal(t, 200, login("new-password"), "the new password must work without a restart")

	req = httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", oldToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.NotEqual(t, 200, parseRespCode(t, w), "a session issued before the reset must be rejected")
}
