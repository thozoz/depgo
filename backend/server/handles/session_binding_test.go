package handles_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loginWithClient signs in as a specific device, the way a browser or CLI does.
func loginWithClient(t *testing.T, handler http.Handler, username, password, clientID string) string {
	t.Helper()
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Id", clientID)
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

func TestBoundSessionCannotBeRecreatedAfterExpiryOrCleanup(t *testing.T) {
	for _, state := range []string{"expired", "evicted-and-cleaned", "missing"} {
		for _, route := range []string{"/api/me", "/d/private.txt", "/p/private.txt"} {
			t.Run(state+route, func(t *testing.T) {
				r, _, _, _, _ := setupTestServer(t)
				seedTestRole(t)
				user := &model.User{ID: 710, Username: "expiry-user", Role: model.Roles{testRoleID}, BasePath: "/", Authn: "[]"}
				user.SetPassword("expiry-password")
				require.NoError(t, db.CreateUser(user))
				token := loginWithClient(t, r, user.Username, "expiry-password", "expiry-device")
				key := utils.GetMD5EncodeStr(fmt.Sprintf("%d-expiry-device", user.ID))
				require.NoError(t, db.UpdateSessionLastActive(user.ID, key, 1))
				if state == "evicted-and-cleaned" {
					require.NoError(t, db.MarkInactive(key))
					require.NoError(t, db.DeleteSessionsBefore(time.Now().Unix()))
				} else if state == "missing" {
					require.NoError(t, db.DeleteSession(user.ID, key))
				}
				req := httptest.NewRequest("GET", route, nil)
				req.Header.Set("Authorization", token)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				assert.Equal(t, 401, parseRespCode(t, w), w.Body.String())
				sess, err := db.GetSession(user.ID, key)
				if state == "expired" {
					require.NoError(t, err)
					assert.Equal(t, int64(1), sess.LastActive)
				} else {
					assert.Error(t, err, "a bearer request must not enroll a missing session")
				}
			})
		}
	}
}

func TestSessionMetadataRefreshCannotUndoEviction(t *testing.T) {
	setupTestServer(t)
	key := "refresh-evicted"
	require.NoError(t, db.CreateSession(&model.Session{UserID: 1, DeviceKey: key, Status: model.SessionActive}))
	require.NoError(t, db.MarkInactive(key))
	assert.Error(t, db.RefreshActiveSession(1, key, "agent", "masked-ip", time.Now().Unix()))
	sess, err := db.GetSession(1, key)
	require.NoError(t, err)
	assert.Equal(t, model.SessionInactive, sess.Status)
}

func meRequest(handler http.Handler, token, clientID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/api/me", nil)
	req.Header.Set("Authorization", token)
	if clientID != "" {
		req.Header.Set("Client-Id", clientID)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w
}

func sessionsOf(t *testing.T, userID uint) []model.Session {
	t.Helper()
	var sessions []model.Session
	require.NoError(t, db.GetDb().Where("user_id = ?", userID).Find(&sessions).Error)
	return sessions
}

// TestEvictedSessionCannotBeResumedWithAnotherClientID is the regression test for
// session deactivation that only checked a client-supplied header. Evicting a
// device marked one row inactive, but the token was not bound to that row: the
// same token with a changed or omitted Client-Id hashed to a different key, no
// row was found, and a fresh active session was created, so the eviction did
// nothing.
func TestEvictedSessionCannotBeResumedWithAnotherClientID(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 700, Username: "sessionuser", Role: model.Roles{testRoleID}, Permission: 0x1, BasePath: "/", Authn: "[]"}
	user.SetPassword("session-password")
	require.NoError(t, db.CreateUser(user))

	const device = "device-A"
	token := loginWithClient(t, r, "sessionuser", "session-password", device)
	key := utils.GetMD5EncodeStr(fmt.Sprintf("%d-%s", user.ID, device))

	// The token must actually work before it is retired.
	require.Equal(t, 200, parseRespCode(t, meRequest(r, token, device)))

	// Evict that device, as POST /api/me/sessions/evict does.
	require.NoError(t, db.MarkInactive(key))

	for _, clientID := range []string{"device-B", "", device, "completely-different-client"} {
		name := clientID
		if name == "" {
			name = "<omitted>"
		}
		t.Run("client="+name, func(t *testing.T) {
			w := meRequest(r, token, clientID)
			assert.NotEqual(t, 200, parseRespCode(t, w),
				"a deactivated session must not be resumable by sending Client-Id %q, body=%s", clientID, w.Body.String())
		})
	}

	// No new session may have been created by those attempts.
	sessions := sessionsOf(t, user.ID)
	require.Len(t, sessions, 1, "attempts with a stolen token must not create sessions")
	assert.Equal(t, model.SessionInactive, sessions[0].Status, "the evicted session must stay inactive")
}

// TestLiveSessionIgnoresClientIDChanges proves the claim wins in the other
// direction too: a live token keeps using the session it was minted for, so a
// client cannot accumulate extra device slots by varying Client-Id.
func TestLiveSessionIgnoresClientIDChanges(t *testing.T) {
	r, _, _, _, _ := setupTestServer(t)
	seedTestRole(t)

	user := &model.User{ID: 701, Username: "stableuser", Role: model.Roles{testRoleID}, Permission: 0x1, BasePath: "/", Authn: "[]"}
	user.SetPassword("stable-session-password")
	require.NoError(t, db.CreateUser(user))

	const device = "stable-device"
	token := loginWithClient(t, r, "stableuser", "stable-session-password", device)

	for _, clientID := range []string{device, "another-device", ""} {
		w := meRequest(r, token, clientID)
		assert.Equal(t, 200, parseRespCode(t, w),
			"a live session must keep working regardless of Client-Id %q, body=%s", clientID, w.Body.String())
	}

	sessions := sessionsOf(t, user.ID)
	require.Len(t, sessions, 1, "varying Client-Id must not mint additional sessions")
	assert.Equal(t, utils.GetMD5EncodeStr(fmt.Sprintf("%d-%s", user.ID, device)), sessions[0].DeviceKey)
	assert.Equal(t, model.SessionActive, sessions[0].Status)
	assert.WithinDuration(t, time.Now(), time.Unix(sessions[0].LastActive, 0), time.Minute)
}
