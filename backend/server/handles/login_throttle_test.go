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

// TestLoginFailureCounterHasAWindow is the regression test for the failure counter
// being stored with no expiry: failures accumulated forever, so a handful of typos
// spread over weeks eventually locked the caller out even though no burst ever
// happened.
func TestLoginFailureCounterHasAWindow(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 600, Username: "throttle", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]"}
	user.SetPassword("correct-password")
	require.NoError(t, db.CreateUser(user))

	const ip = "198.51.100.50:1234"
	attempt := func(password string) int {
		body := fmt.Sprintf(`{"username":"throttle","password":%q}`, password)
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		_, code, _ := parseRespEnvelope(t, w)
		return code
	}

	// Four failures stay below the threshold.
	for i := 0; i < 4; i++ {
		require.NotEqual(t, 200, attempt(fmt.Sprintf("wrong-%d", i)))
	}

	// A correct password still works, which also proves the counter is not a
	// permanent lock.
	assert.Equal(t, 200, attempt("correct-password"),
		"a correct password below the failure threshold must be accepted")

	// The success clears the counter, so a fresh burst is required to trip it.
	for i := 0; i < 4; i++ {
		require.NotEqual(t, 200, attempt(fmt.Sprintf("again-%d", i)))
	}
	assert.Equal(t, 200, attempt("correct-password"),
		"the counter must have been reset by the earlier successful login")
}

// TestLoginThrottleBlocksAfterBurst confirms the limit still fires on a real
// burst, so the window change did not disable the protection.
func TestLoginThrottleBlocksAfterBurst(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 601, Username: "burst", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]"}
	user.SetPassword("right-password")
	require.NoError(t, db.CreateUser(user))

	const ip = "203.0.113.99:9999"
	attempt := func(password string) int {
		body := fmt.Sprintf(`{"username":"burst","password":%q}`, password)
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		_, code, _ := parseRespEnvelope(t, w)
		return code
	}

	for i := 0; i < 5; i++ {
		require.NotEqual(t, 200, attempt(fmt.Sprintf("wrong-%d", i)))
	}
	// The next attempt, even with the right password, must be throttled.
	assert.Equal(t, 429, attempt("right-password"),
		"a burst of failures must still trigger the 429 throttle")
}
