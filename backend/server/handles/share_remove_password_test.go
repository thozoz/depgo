package handles_test

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An empty password on update means "keep the current one", so without a
// dedicated flag there was no way to take a password off a share again.
func TestUpdateShareCanRemovePassword(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)
	shareID, _ := createShareFixture(t, r, adminToken, rootDir, "open-later.txt", "temp-pass")

	info := func() (hasPassword, authed bool) {
		req := httptest.NewRequest("GET", "/api/public/share/info?share_id="+shareID, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, 200, parseRespCode(t, w), w.Body.String())
		var resp struct {
			Data struct {
				HasPassword bool `json:"has_password"`
				Authed      bool `json:"authed"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		return resp.Data.HasPassword, resp.Data.Authed
	}
	update := func(body string) {
		req := httptest.NewRequest("POST", "/api/share/update", strings.NewReader(body))
		req.Header.Set("Authorization", adminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, 200, parseRespCode(t, w), w.Body.String())
	}

	hasPassword, authed := info()
	require.True(t, hasPassword)
	require.False(t, authed)

	// Leaving the password empty keeps it.
	update(fmt.Sprintf(`{"share_id":"%s","name":"renamed"}`, shareID))
	hasPassword, _ = info()
	assert.True(t, hasPassword, "an empty password must keep the current one")

	// The explicit flag removes it.
	update(fmt.Sprintf(`{"share_id":"%s","remove_password":true}`, shareID))
	hasPassword, authed = info()
	assert.False(t, hasPassword, "remove_password must drop the protection")
	assert.True(t, authed, "a share without a password is open to anyone with the link")

}
