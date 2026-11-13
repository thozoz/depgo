package mcp

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupSessionDB initialises an in-memory database for the session checks.
func setupSessionDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, _ := database.DB()
	if sqlDB != nil {
		sqlDB.SetMaxOpenConns(1)
	}
	db.Init(database)
	t.Cleanup(func() {
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
}

func deviceKeyFor(userID uint, clientID string) string {
	return utils.GetMD5EncodeStr(fmt.Sprintf("%d-%s", userID, clientID))
}

// TestRequireActiveSessionAllowsActiveSession covers the normal case.
func TestRequireActiveSessionAllowsActiveSession(t *testing.T) {
	setupSessionDB(t)

	const userID = 400
	const clientID = "mcp-client"
	key := deviceKeyFor(userID, clientID)
	require.NoError(t, db.CreateSession(&model.Session{
		UserID: userID, DeviceKey: key,
		LastActive: time.Now().Unix(), Status: model.SessionActive,
	}))

	assert.NoError(t, requireActiveSession(&model.User{ID: userID}, key))
}

// TestRequireActiveSessionRejectsDeactivated is the regression test for evicted
// sessions still working over MCP: the HTTP middleware noticed, MCP did not.
func TestRequireActiveSessionRejectsDeactivated(t *testing.T) {
	setupSessionDB(t)

	const userID = 401
	const clientID = "evicted-client"
	key := deviceKeyFor(userID, clientID)
	require.NoError(t, db.CreateSession(&model.Session{
		UserID: userID, DeviceKey: key,
		LastActive: time.Now().Unix(), Status: model.SessionActive,
	}))
	require.NoError(t, db.MarkInactive(key))

	err := requireActiveSession(&model.User{ID: userID}, key)
	require.Error(t, err, "a deactivated session must be refused over MCP")
	assert.Contains(t, err.Error(), "deactivated")
}

// TestRequireActiveSessionAllowsUnknownSession documents the deliberate leniency:
// an MCP client that never logged in over HTTP has no session row, and demanding
// one would lock out legitimate automation.
func TestRequireActiveSessionAllowsUnknownSession(t *testing.T) {
	setupSessionDB(t)

	assert.NoError(t, requireActiveSession(&model.User{ID: 402}, deviceKeyFor(402, "never-logged-in")))
}

// TestRequireActiveSessionIgnoresClientIDHeader is the regression test for
// deactivation that could be bypassed by changing the Client-Id header: the
// session key must come from the signed token claim, not from the request.
func TestRequireActiveSessionIgnoresClientIDHeader(t *testing.T) {
	setupSessionDB(t)

	const userID = 403
	const clientID = "query-client"
	key := deviceKeyFor(userID, clientID)
	require.NoError(t, db.CreateSession(&model.Session{
		UserID: userID, DeviceKey: key,
		LastActive: time.Now().Unix(), Status: model.SessionActive,
	}))
	require.NoError(t, db.MarkInactive(key))

	// The credential is bound to the retired key; a request cannot talk its way
	// out of it by naming another client.
	assert.Error(t, requireActiveSession(&model.User{ID: userID}, key))
}

func TestHTTPContextRejectsEvictedTokenWithChangedClientID(t *testing.T) {
	setupSessionDB(t)
	oldConf := conf.Conf
	conf.Conf = conf.DefaultConfig()
	t.Cleanup(func() { conf.Conf = oldConf })
	user := &model.User{ID: 404, Username: "bound-mcp-user"}
	require.NoError(t, db.CreateUser(user))
	key := deviceKeyFor(user.ID, "original")
	require.NoError(t, db.CreateSession(&model.Session{UserID: user.ID, DeviceKey: key, Status: model.SessionActive, LastActive: time.Now().Unix()}))
	token, err := common.GenerateTokenWithDeviceKey(user, key)
	require.NoError(t, err)
	t.Cleanup(func() { _ = common.InvalidateToken(token) })

	request := func(clientID string) context.Context {
		r := httptest.NewRequest("POST", "/mcp?client_id="+clientID, nil)
		r.Header.Set("Authorization", token)
		r.Header.Set("Client-Id", clientID)
		return HTTPContextFunc(context.Background(), r)
	}
	_, err = resolveUser(request("original"))
	require.NoError(t, err)
	require.NoError(t, db.MarkInactive(key))
	for _, clientID := range []string{"different", "", "original"} {
		_, err := resolveUser(request(clientID))
		assert.Error(t, err, "client id %q must not bypass eviction", clientID)
	}
}

func TestSessionCheckFailsClosedOnDatabaseError(t *testing.T) {
	setupSessionDB(t)
	sqlDB, err := db.GetDb().DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	assert.Error(t, requireActiveSession(&model.User{ID: 405}, "bound-key"))
}

// TestRequireActiveSessionExemptsSessionlessCredentials proves the site admin
// token and the guest credential, which are not device sessions, are not blocked.
func TestRequireActiveSessionExemptsSessionlessCredentials(t *testing.T) {
	setupSessionDB(t)
	assert.NoError(t, requireActiveSession(&model.User{ID: 1}, ""))
}
