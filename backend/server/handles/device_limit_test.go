package handles_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setMaxDevices configures the device policy for a test.
func setMaxDevices(t *testing.T, max int, policy string) {
	t.Helper()
	require.NoError(t, op.SaveSettingItem(&model.SettingItem{
		Key: conf.MaxDevices, Value: fmt.Sprint(max), Type: conf.TypeNumber, Group: model.GLOBAL, Flag: model.PRIVATE,
	}))
	require.NoError(t, op.SaveSettingItem(&model.SettingItem{
		Key: conf.DeviceEvictPolicy, Value: policy, Type: conf.TypeSelect, Group: model.GLOBAL, Flag: model.PRIVATE,
	}))
	op.SettingCacheUpdate()
	t.Cleanup(func() {
		_ = op.SaveSettingItem(&model.SettingItem{Key: conf.MaxDevices, Value: "0", Type: conf.TypeNumber, Group: model.GLOBAL, Flag: model.PRIVATE})
		_ = op.SaveSettingItem(&model.SettingItem{Key: conf.DeviceEvictPolicy, Value: "deny", Type: conf.TypeSelect, Group: model.GLOBAL, Flag: model.PRIVATE})
		op.SettingCacheUpdate()
	})
}

// loginFromDevice performs a login pretending to be a distinct client device.
func loginFromDevice(t *testing.T, handler http.Handler, username, password, clientID string) (status int, bodyCode int) {
	t.Helper()
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Id", clientID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp struct {
		Code int `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp.Code
}

// TestMaxDevicesEnforcedUnderConcurrency is the regression test for a limit that
// could be overshot simply by logging in from several clients at once: the
// budget was counted and the session created in two separate statements.
func TestMaxDevicesEnforcedUnderConcurrency(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)
	setMaxDevices(t, 2, "deny")

	user := &model.User{ID: 300, Username: "devrace", Role: model.Roles{testRoleID}, Permission: 0x1, BasePath: "/", Authn: "[]"}
	user.SetPassword("devrace-password")
	require.NoError(t, db.CreateUser(user))

	const attempts = 12
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			loginFromDevice(t, r, "devrace", "devrace-password", fmt.Sprintf("client-%d", i))
		}(i)
	}
	wg.Wait()

	active, err := db.CountActiveSessionsByUser(300)
	require.NoError(t, err)
	assert.LessOrEqual(t, active, int64(2),
		"max_devices=2 must hold under concurrent logins, got %d active sessions", active)
	assert.Greater(t, active, int64(0), "at least one login should have succeeded")
}

// TestMaxDevicesAllowedSequentially checks the limit still admits exactly the
// configured number of devices and then refuses.
func TestMaxDevicesAllowedSequentially(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)
	setMaxDevices(t, 2, "deny")

	user := &model.User{ID: 301, Username: "devseq", Role: model.Roles{testRoleID}, Permission: 0x1, BasePath: "/", Authn: "[]"}
	user.SetPassword("devseq-password")
	require.NoError(t, db.CreateUser(user))

	for i := 0; i < 2; i++ {
		_, code := loginFromDevice(t, r, "devseq", "devseq-password", fmt.Sprintf("dev-%d", i))
		require.Equal(t, 200, code, "device %d should be allowed", i+1)
	}
	_, code := loginFromDevice(t, r, "devseq", "devseq-password", "dev-overflow")
	assert.NotEqual(t, 200, code, "the third device must be refused when the limit is 2")

	active, err := db.CountActiveSessionsByUser(301)
	require.NoError(t, err)
	assert.Equal(t, int64(2), active)
}

// TestMaxDevicesEvictOldestPolicyKeepsLoginsWorking proves the evict_oldest
// policy still works after the fix: the third device retires the oldest instead
// of being refused, and the ceiling is respected.
func TestMaxDevicesEvictOldestPolicyKeepsLoginsWorking(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)
	setMaxDevices(t, 2, "evict_oldest")

	user := &model.User{ID: 302, Username: "devevict", Role: model.Roles{testRoleID}, Permission: 0x1, BasePath: "/", Authn: "[]"}
	user.SetPassword("devevict-password")
	require.NoError(t, db.CreateUser(user))

	for i := 0; i < 4; i++ {
		_, code := loginFromDevice(t, r, "devevict", "devevict-password", fmt.Sprintf("evict-%d", i))
		require.Equal(t, 200, code, "evict_oldest must let device %d in", i+1)

		active, err := db.CountActiveSessionsByUser(302)
		require.NoError(t, err)
		assert.LessOrEqual(t, active, int64(2),
			"after device %d the active count must stay within the limit", i+1)
	}
}
