package handles_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResetTokenPreservesSettingMetadata is the regression test for the site
// token reset clobbering the setting row. The handler used to build a fresh
// SettingItem carrying only key and value and save it, which wrote zeroes into
// every other column: the row lost its display index, its type and its recorded
// default, and the admin settings page showed it in the wrong place.
func TestResetTokenPreservesSettingMetadata(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)

	// Give the token row the metadata a real installation has.
	item, err := op.GetSettingItemByKey(conf.Token)
	require.NoError(t, err)
	item.Index = 42
	item.Help = "admin token used for authentication"
	item.Type = conf.TypeString
	item.Flag = 1 // PRIVATE
	require.NoError(t, op.SaveSettingItem(item))

	req := httptest.NewRequest("POST", "/api/admin/setting/reset_token", nil)
	req.Header.Set("Authorization", adminToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "reset_token failed: %s", w.Body.String())

	var resp struct {
		Data string `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Data, "the new token must be returned")

	after, err := db.GetSettingItemByKey(conf.Token)
	require.NoError(t, err)
	assert.Equal(t, resp.Data, after.Value, "the stored token must be the one that was returned")
	assert.NotEqual(t, "depgo-test-admin-token", after.Value, "the token must actually change")

	assert.Equal(t, uint(42), after.Index, "reset must not reset the display index")
	assert.Equal(t, "admin token used for authentication", after.Help, "reset must not clear the help text")
	assert.Equal(t, conf.TypeString, after.Type, "reset must not clear the value type")
	assert.Equal(t, 1, after.Flag, "reset must not change the visibility flag")
}

// TestSetTokenPreservesSettingMetadata covers the explicit set path, which had
// the same flaw.
func TestSetTokenPreservesSettingMetadata(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)

	item, err := op.GetSettingItemByKey(conf.Token)
	require.NoError(t, err)
	item.Index = 7
	item.Help = "keep me"
	require.NoError(t, op.SaveSettingItem(item))

	req := httptest.NewRequest("POST", "/api/admin/setting/set_token", strings.NewReader(`{"token":"chosen-by-operator-token"}`))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "set_token failed: %s", w.Body.String())

	after, err := db.GetSettingItemByKey(conf.Token)
	require.NoError(t, err)
	assert.Equal(t, "chosen-by-operator-token", after.Value)
	assert.Equal(t, uint(7), after.Index)
	assert.Equal(t, "keep me", after.Help)
}

// TestSetTokenChangesWhatTheServerAccepts proves the new token takes effect: the
// old admin token stops working and the new one is accepted.
func TestSetTokenChangesWhatTheServerAccepts(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)

	req := httptest.NewRequest("POST", "/api/admin/setting/set_token", strings.NewReader(`{"token":"rotated-admin-token"}`))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 200, parseRespCode(t, w), "set_token failed: %s", w.Body.String())

	// Old token must be refused now.
	oldReq := httptest.NewRequest("GET", "/api/me", nil)
	oldReq.Header.Set("Authorization", adminToken)
	wOld := httptest.NewRecorder()
	r.ServeHTTP(wOld, oldReq)
	_, oldCode, _ := parseRespEnvelope(t, wOld)
	assert.NotEqual(t, 200, oldCode, "the previous admin token must stop working")

	// New token must be accepted.
	newReq := httptest.NewRequest("GET", "/api/me", nil)
	newReq.Header.Set("Authorization", "rotated-admin-token")
	wNew := httptest.NewRecorder()
	r.ServeHTTP(wNew, newReq)
	_, newCode, msg := parseRespEnvelope(t, wNew)
	assert.Equal(t, 200, newCode, "the new admin token must work, msg=%q", msg)
}
