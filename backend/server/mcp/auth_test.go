package mcp

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils"
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
	require.NoError(t, db.CreateSession(&model.Session{
		UserID: userID, DeviceKey: deviceKeyFor(userID, clientID),
		LastActive: time.Now().Unix(), Status: model.SessionActive,
	}))

	req := httptest.NewRequest("GET", "/mcp", nil)
	req.Header.Set("Client-Id", clientID)

	assert.NoError(t, requireActiveSession(req, &model.User{ID: userID}))
}

// TestRequireActiveSessionRejectsDeactivated is the regression test for evicted
// sessions still working over MCP: the HTTP middleware noticed, MCP did not.
func TestRequireActiveSessionRejectsDeactivated(t *testing.T) {
	setupSessionDB(t)

	const userID = 401
	const clientID = "evicted-client"
	require.NoError(t, db.CreateSession(&model.Session{
		UserID: userID, DeviceKey: deviceKeyFor(userID, clientID),
		LastActive: time.Now().Unix(), Status: model.SessionActive,
	}))
	require.NoError(t, db.MarkInactive(deviceKeyFor(userID, clientID)))

	req := httptest.NewRequest("GET", "/mcp", nil)
	req.Header.Set("Client-Id", clientID)

	err := requireActiveSession(req, &model.User{ID: userID})
	require.Error(t, err, "a deactivated session must be refused over MCP")
	assert.Contains(t, err.Error(), "deactivated")
}

// TestRequireActiveSessionAllowsUnknownSession documents the deliberate leniency:
// an MCP client that never logged in over HTTP has no session row, and demanding
// one would lock out legitimate automation.
func TestRequireActiveSessionAllowsUnknownSession(t *testing.T) {
	setupSessionDB(t)

	req := httptest.NewRequest("GET", "/mcp", nil)
	assert.NoError(t, requireActiveSession(req, &model.User{ID: 402}))
}

// TestRequireActiveSessionReadsClientIDFromQuery mirrors the middleware, which
// accepts client_id as a query parameter as well.
func TestRequireActiveSessionReadsClientIDFromQuery(t *testing.T) {
	setupSessionDB(t)

	const userID = 403
	const clientID = "query-client"
	require.NoError(t, db.CreateSession(&model.Session{
		UserID: userID, DeviceKey: deviceKeyFor(userID, clientID),
		LastActive: time.Now().Unix(), Status: model.SessionActive,
	}))
	require.NoError(t, db.MarkInactive(deviceKeyFor(userID, clientID)))

	req := httptest.NewRequest("GET", "/mcp?client_id="+clientID, nil)
	require.Error(t, requireActiveSession(req, &model.User{ID: userID}))
}
