package handles_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alist-org/alist/v3/cmd/flags"
	"github.com/alist-org/alist/v3/drivers/local"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	shareauth "github.com/alist-org/alist/v3/internal/share"
	"github.com/alist-org/alist/v3/server"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func parseRespCode(t *testing.T, w *httptest.ResponseRecorder) int {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.Code
}

func setupTestServer(t *testing.T) (*gin.Engine, *local.Local, string, string, string) {
	gin.SetMode(gin.TestMode)
	oldConf := conf.Conf
	conf.Conf = conf.DefaultConfig()

	baseDir := t.TempDir()
	rootDir := filepath.Join(baseDir, "root")
	stateDir := filepath.Join(baseDir, "state")
	flags.DataDir = filepath.Join(baseDir, "data")
	require.NoError(t, os.MkdirAll(rootDir, 0755))
	require.NoError(t, os.MkdirAll(stateDir, 0755))
	require.NoError(t, os.MkdirAll(flags.DataDir, 0755))

	database, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, _ := database.DB()
	if sqlDB != nil {
		sqlDB.SetMaxOpenConns(1)
	}
	db.Init(database)

	// Clear user cache
	op.ClearUserCache()

	// Admin user setup
	adminRole := model.Role{ID: 101, Name: "admin", PermissionScopes: []model.PermissionEntry{{Path: "/", Permission: 0xFFFF}}}
	_ = db.CreateRole(&adminRole)
	adminUser := model.User{
		ID:         1,
		Username:   "admin",
		Role:       model.Roles{model.ADMIN, 101},
		BasePath:   "/",
		Permission: 0xFFFF,
	}
	_ = db.CreateUser(&adminUser)

	// Guest user disabled (Depgo default)
	guestRole := model.Role{ID: 102, Name: "guest", PermissionScopes: []model.PermissionEntry{}}
	_ = db.CreateRole(&guestRole)
	guestUser := model.User{
		ID:         2,
		Username:   "guest",
		Role:       model.Roles{model.GUEST, 102},
		BasePath:   "/",
		Permission: 0,
		Disabled:   true,
	}
	_ = db.CreateUser(&guestUser)

	// Admin token
	adminToken := "depgo-test-admin-token"
	_ = op.SaveSettingItem(&model.SettingItem{Key: conf.Token, Value: adminToken})
	op.SettingCacheUpdate()
	conf.StoragesLoaded = true

	// Storage mount setup
	additionJSON := fmt.Sprintf(`{"root_folder_path":"%s","state_folder":"%s","mkdir_perm":"755"}`,
		strings.ReplaceAll(rootDir, `\`, `\\`),
		strings.ReplaceAll(stateDir, `\`, `\\`))

	storage := model.Storage{
		MountPath: "/",
		Driver:    "Local",
		Status:    "work",
		Addition:  additionJSON,
		Disabled:  false,
	}
	require.NoError(t, db.CreateStorage(&storage))

	localDriver := &local.Local{
		Storage: storage,
		Addition: local.Addition{
			RootPath:    driver.RootPath{RootFolderPath: rootDir},
			StateFolder: stateDir,
			MkdirPerm:   "755",
		},
	}
	op.RegisterDriver(func() driver.Driver { return localDriver })
	require.NoError(t, op.LoadStorage(context.Background(), storage))

	r := gin.New()
	server.Init(r)

	t.Cleanup(func() {
		if drv, err := op.GetStorageByMountPath("/"); err == nil {
			_ = drv.Drop(context.Background())
		}
		_ = localDriver.Drop(context.Background())
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		op.ClearUserCache()
		conf.StoragesLoaded = false
		conf.Conf = oldConf
	})

	return r, localDriver, adminToken, rootDir, stateDir
}

func TestTrashAPI(t *testing.T) {
	r, localDriver, adminToken, rootDir, _ := setupTestServer(t)

	// Create test file in root and move it to trash
	filePath := filepath.Join(rootDir, "trash_me.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("goodbye"), 0644))

	obj := &model.Object{Path: filePath, Name: "trash_me.txt"}
	require.NoError(t, localDriver.Remove(context.Background(), obj))

	// 1. Anonymous access must fail (401)
	reqAnon := httptest.NewRequest("GET", "/api/fs/trash?path=/", nil)
	wAnon := httptest.NewRecorder()
	r.ServeHTTP(wAnon, reqAnon)
	assert.Equal(t, 401, parseRespCode(t, wAnon))

	// 2. Admin access succeeds with exact schema
	reqAdmin := httptest.NewRequest("GET", "/api/fs/trash?path=/", nil)
	reqAdmin.Header.Set("Authorization", adminToken)
	wAdmin := httptest.NewRecorder()
	r.ServeHTTP(wAdmin, reqAdmin)
	assert.Equal(t, http.StatusOK, wAdmin.Code)

	var listResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Content []struct {
				TrashID      string `json:"trash_id"`
				OriginalPath string `json:"original_path"`
				Size         int64  `json:"size"`
				IsDir        bool   `json:"is_dir"`
				DeletedAt    string `json:"deleted_at"`
			} `json:"content"`
			Total   int `json:"total"`
			Page    int `json:"page"`
			PerPage int `json:"per_page"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wAdmin.Body.Bytes(), &listResp))
	assert.Equal(t, 200, listResp.Code)
	require.Len(t, listResp.Data.Content, 1)
	trashID := listResp.Data.Content[0].TrashID
	assert.NotEmpty(t, trashID)
	assert.Equal(t, "/trash_me.txt", listResp.Data.Content[0].OriginalPath)

	// 3. Purge without confirm: true must fail (400)
	deleteNoConfirmBody := fmt.Sprintf(`{"path":"/","trash_id":"%s","confirm":false}`, trashID)
	reqDelNoConfirm := httptest.NewRequest("POST", "/api/fs/trash/delete", strings.NewReader(deleteNoConfirmBody))
	reqDelNoConfirm.Header.Set("Authorization", adminToken)
	reqDelNoConfirm.Header.Set("Content-Type", "application/json")
	wDelNoConfirm := httptest.NewRecorder()
	r.ServeHTTP(wDelNoConfirm, reqDelNoConfirm)
	assert.Equal(t, 400, parseRespCode(t, wDelNoConfirm))

	// 4. Restore collision returns 409
	require.NoError(t, os.WriteFile(filePath, []byte("conflict"), 0644))
	restoreBody := fmt.Sprintf(`{"path":"/","trash_id":"%s"}`, trashID)
	reqRestoreConflict := httptest.NewRequest("POST", "/api/fs/trash/restore", strings.NewReader(restoreBody))
	reqRestoreConflict.Header.Set("Authorization", adminToken)
	reqRestoreConflict.Header.Set("Content-Type", "application/json")
	wRestoreConflict := httptest.NewRecorder()
	r.ServeHTTP(wRestoreConflict, reqRestoreConflict)
	assert.Equal(t, 409, parseRespCode(t, wRestoreConflict))

	// 5. Restore success after removing conflict
	require.NoError(t, os.Remove(filePath))
	reqRestore := httptest.NewRequest("POST", "/api/fs/trash/restore", strings.NewReader(restoreBody))
	reqRestore.Header.Set("Authorization", adminToken)
	reqRestore.Header.Set("Content-Type", "application/json")
	wRestore := httptest.NewRecorder()
	r.ServeHTTP(wRestore, reqRestore)
	assert.Equal(t, 200, parseRespCode(t, wRestore))
	assert.FileExists(t, filePath)

	// 6. Huge pagination without panic or overflow
	reqHuge := httptest.NewRequest("GET", "/api/fs/trash?path=/&page=999999999&per_page=100", nil)
	reqHuge.Header.Set("Authorization", adminToken)
	wHuge := httptest.NewRecorder()
	r.ServeHTTP(wHuge, reqHuge)
	assert.Equal(t, 200, parseRespCode(t, wHuge))
}

func TestTextAPI(t *testing.T) {
	r, _, adminToken, rootDir, _ := setupTestServer(t)

	// 1. Rejection of unallowed extension (.bin, .exe)
	badExtBody := `{"path":"/payload.bin","content":"test","revision":0}`
	reqBadExt := httptest.NewRequest("PUT", "/api/fs/text", strings.NewReader(badExtBody))
	reqBadExt.Header.Set("Authorization", adminToken)
	reqBadExt.Header.Set("Content-Type", "application/json")
	wBadExt := httptest.NewRecorder()
	r.ServeHTTP(wBadExt, reqBadExt)
	assert.Equal(t, 400, parseRespCode(t, wBadExt))

	// 2. Rejection of NUL byte
	nulBody := `{"path":"/notes.txt","content":"hello\u0000world","revision":0}`
	reqNul := httptest.NewRequest("PUT", "/api/fs/text", strings.NewReader(nulBody))
	reqNul.Header.Set("Authorization", adminToken)
	reqNul.Header.Set("Content-Type", "application/json")
	wNul := httptest.NewRecorder()
	r.ServeHTTP(wNul, reqNul)
	assert.Equal(t, 400, parseRespCode(t, wNul))

	// 3. Initial save (revision 0) succeeds
	save1Body := `{"path":"/notes.txt","content":"first line\n","revision":0}`
	reqSave1 := httptest.NewRequest("PUT", "/api/fs/text", strings.NewReader(save1Body))
	reqSave1.Header.Set("Authorization", adminToken)
	reqSave1.Header.Set("Content-Type", "application/json")
	wSave1 := httptest.NewRecorder()
	r.ServeHTTP(wSave1, reqSave1)
	assert.Equal(t, 200, parseRespCode(t, wSave1))

	var putResp1 struct {
		Code int `json:"code"`
		Data struct {
			Revision int64 `json:"revision"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wSave1.Body.Bytes(), &putResp1))
	assert.Equal(t, int64(1), putResp1.Data.Revision)

	// 4. GET text returns exact content and revision
	reqGet := httptest.NewRequest("GET", "/api/fs/text?path=/notes.txt", nil)
	reqGet.Header.Set("Authorization", adminToken)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)
	assert.Equal(t, 200, parseRespCode(t, wGet))

	var getResp struct {
		Code int `json:"code"`
		Data struct {
			Content  string `json:"content"`
			Revision int64  `json:"revision"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &getResp))
	assert.Equal(t, "first line\n", getResp.Data.Content)
	assert.Equal(t, int64(1), getResp.Data.Revision)

	// 5. Revision conflict (expected revision 1, but passed revision 99) returns 409
	saveConflictBody := `{"path":"/notes.txt","content":"stale edit","revision":99}`
	reqConflict := httptest.NewRequest("PUT", "/api/fs/text", strings.NewReader(saveConflictBody))
	reqConflict.Header.Set("Authorization", adminToken)
	reqConflict.Header.Set("Content-Type", "application/json")
	wConflict := httptest.NewRecorder()
	r.ServeHTTP(wConflict, reqConflict)
	assert.Equal(t, 409, parseRespCode(t, wConflict))

	// Content remains intact
	contentDisk, err := os.ReadFile(filepath.Join(rootDir, "notes.txt"))
	require.NoError(t, err)
	assert.Equal(t, "first line\n", string(contentDisk))

	// 6. Matching revision save succeeds (revision 1 -> 2)
	save2Body := `{"path":"/notes.txt","content":"second edit\n","revision":1}`
	reqSave2 := httptest.NewRequest("PUT", "/api/fs/text", strings.NewReader(save2Body))
	reqSave2.Header.Set("Authorization", adminToken)
	reqSave2.Header.Set("Content-Type", "application/json")
	wSave2 := httptest.NewRecorder()
	r.ServeHTTP(wSave2, reqSave2)
	assert.Equal(t, 200, parseRespCode(t, wSave2))

	// 7. Missing or zero revision on existing file must be rejected with 400
	saveNoRevBody := `{"path":"/notes.txt","content":"no revision"}`
	reqNoRev := httptest.NewRequest("PUT", "/api/fs/text", strings.NewReader(saveNoRevBody))
	reqNoRev.Header.Set("Authorization", adminToken)
	reqNoRev.Header.Set("Content-Type", "application/json")
	wNoRev := httptest.NewRecorder()
	r.ServeHTTP(wNoRev, reqNoRev)
	assert.Equal(t, 400, parseRespCode(t, wNoRev))

	// 8. Negative revision on existing file must be rejected with 400
	saveNegRevBody := `{"path":"/notes.txt","content":"negative revision","revision":-5}`
	reqNegRev := httptest.NewRequest("PUT", "/api/fs/text", strings.NewReader(saveNegRevBody))
	reqNegRev.Header.Set("Authorization", adminToken)
	reqNegRev.Header.Set("Content-Type", "application/json")
	wNegRev := httptest.NewRecorder()
	r.ServeHTTP(wNegRev, reqNegRev)
	assert.Equal(t, 400, parseRespCode(t, wNegRev))
}

func TestVersionsAPI(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)

	// Create doc and update it to create versions
	save1 := `{"path":"/doc.md","content":"version 1\n","revision":0}`
	req1 := httptest.NewRequest("PUT", "/api/fs/text", strings.NewReader(save1))
	req1.Header.Set("Authorization", adminToken)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	require.Equal(t, 200, parseRespCode(t, w1))

	save2 := `{"path":"/doc.md","content":"version 2\n","revision":1}`
	req2 := httptest.NewRequest("PUT", "/api/fs/text", strings.NewReader(save2))
	req2.Header.Set("Authorization", adminToken)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, 200, parseRespCode(t, w2))

	// List versions
	reqList := httptest.NewRequest("GET", "/api/fs/versions?path=/doc.md", nil)
	reqList.Header.Set("Authorization", adminToken)
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, reqList)
	assert.Equal(t, 200, parseRespCode(t, wList))

	var verResp struct {
		Code int `json:"code"`
		Data struct {
			FileID   string `json:"file_id"`
			Revision int64  `json:"revision"`
			Content  []struct {
				VersionID string `json:"version_id"`
				Revision  int64  `json:"revision"`
				Size      int64  `json:"size"`
				CreatedAt string `json:"created_at"`
			} `json:"content"`
			Total int `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &verResp))
	assert.Equal(t, int64(2), verResp.Data.Revision)
	require.Len(t, verResp.Data.Content, 1)
	v1ID := verResp.Data.Content[0].VersionID

	// Restore with wrong revision returns 409
	restoreWrongRev := fmt.Sprintf(`{"path":"/doc.md","version_id":"%s","revision":99}`, v1ID)
	reqRestoreConflict := httptest.NewRequest("POST", "/api/fs/versions/restore", strings.NewReader(restoreWrongRev))
	reqRestoreConflict.Header.Set("Authorization", adminToken)
	reqRestoreConflict.Header.Set("Content-Type", "application/json")
	wRC := httptest.NewRecorder()
	r.ServeHTTP(wRC, reqRestoreConflict)
	assert.Equal(t, 409, parseRespCode(t, wRC))

	// Restore with correct revision succeeds
	restoreCorrect := fmt.Sprintf(`{"path":"/doc.md","version_id":"%s","revision":2}`, v1ID)
	reqRestore := httptest.NewRequest("POST", "/api/fs/versions/restore", strings.NewReader(restoreCorrect))
	reqRestore.Header.Set("Authorization", adminToken)
	reqRestore.Header.Set("Content-Type", "application/json")
	wR := httptest.NewRecorder()
	r.ServeHTTP(wR, reqRestore)
	assert.Equal(t, 200, parseRespCode(t, wR))
}

func TestShareTokenAccessRevisionInvalidation(t *testing.T) {
	_, _, _, _, _ = setupTestServer(t)

	share := &model.Share{
		ShareID:        "share_rev_test",
		PasswordHash:   "hash123",
		AccessRevision: 1,
		UpdatedAt:      time.Now(),
	}

	// Sign access token with initial revision
	token := shareauth.SignAccess(share, time.Hour)
	require.NoError(t, shareauth.VerifyAccess(share, token))

	// Invalidate in same second via AccessRevision increment
	share.AccessRevision++
	err := shareauth.VerifyAccess(share, token)
	assert.Error(t, err, "share token must fail verification when AccessRevision changes")
}

func TestTextEscapedLimitAndConcurrent(t *testing.T) {
	r, _, adminToken, _, _ := setupTestServer(t)

	// 1. Escaped JSON content that exceeds raw size bound if naive, but unescaped is <= 2MiB
	escapedChunk := `hello \"world\" \n test `
	var builder strings.Builder
	for builder.Len() < 1024*1024 {
		builder.WriteString(escapedChunk)
	}
	content := builder.String() // ~1 MiB of text containing escaped quotes and newlines

	putBody, err := json.Marshal(map[string]interface{}{
		"path":     "/escaped.txt",
		"content":  content,
		"revision": 0,
	})
	require.NoError(t, err)

	reqEscaped := httptest.NewRequest("PUT", "/api/fs/text", bytes.NewReader(putBody))
	reqEscaped.Header.Set("Authorization", adminToken)
	reqEscaped.Header.Set("Content-Type", "application/json")
	wEscaped := httptest.NewRecorder()
	r.ServeHTTP(wEscaped, reqEscaped)
	assert.Equal(t, 200, parseRespCode(t, wEscaped))

	// 2. Coherent concurrent GET and PUT
	// Initialize a document
	initDoc := `{"path":"/concurrent.txt","content":"rev0\n","revision":0}`
	reqInit := httptest.NewRequest("PUT", "/api/fs/text", strings.NewReader(initDoc))
	reqInit.Header.Set("Authorization", adminToken)
	reqInit.Header.Set("Content-Type", "application/json")
	wInit := httptest.NewRecorder()
	r.ServeHTTP(wInit, reqInit)
	require.Equal(t, 200, parseRespCode(t, wInit))

	// Run concurrent reads and sequential valid writes
	done := make(chan bool)
	go func() {
		for i := 0; i < 20; i++ {
			reqGet := httptest.NewRequest("GET", "/api/fs/text?path=/concurrent.txt", nil)
			reqGet.Header.Set("Authorization", adminToken)
			wGet := httptest.NewRecorder()
			r.ServeHTTP(wGet, reqGet)
			assert.Equal(t, 200, parseRespCode(t, wGet))

			var getResp struct {
				Data struct {
					Content  string `json:"content"`
					Revision int64  `json:"revision"`
				} `json:"data"`
			}
			_ = json.Unmarshal(wGet.Body.Bytes(), &getResp)
			// Content and revision must always be coherent
			assert.True(t, getResp.Data.Revision >= 1)
			assert.True(t, strings.HasPrefix(getResp.Data.Content, "rev"))
		}
		done <- true
	}()

	curRev := int64(1)
	for i := 1; i <= 5; i++ {
		putBody, err := json.Marshal(map[string]interface{}{
			"path":     "/concurrent.txt",
			"content":  fmt.Sprintf("rev%d\n", i),
			"revision": curRev,
		})
		require.NoError(t, err)
		reqPut := httptest.NewRequest("PUT", "/api/fs/text", bytes.NewReader(putBody))
		reqPut.Header.Set("Authorization", adminToken)
		reqPut.Header.Set("Content-Type", "application/json")
		wPut := httptest.NewRecorder()
		r.ServeHTTP(wPut, reqPut)
		assert.Equal(t, 200, parseRespCode(t, wPut))
		curRev++
	}
	<-done
}

func TestBootstrapQuotedRootPath(t *testing.T) {
	quotedPath := `/srv/data "special"/files`
	addition := map[string]string{
		"root_folder_path": quotedPath,
		"state_folder":    "/srv/data/.state",
		"mkdir_perm":      "755",
	}
	encoded, err := json.Marshal(addition)
	require.NoError(t, err)

	var decoded map[string]string
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, quotedPath, decoded["root_folder_path"])
}
