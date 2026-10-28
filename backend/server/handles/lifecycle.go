package handles

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	stdpath "path"
	"unicode/utf8"

	"github.com/alist-org/alist/v3/drivers/local"
	"github.com/alist-org/alist/v3/internal/localstate"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
)

// parsePaginationSlice computes safe slice bounds for in-memory pagination without integer overflow.
func parsePaginationSlice(page, perPage, total int) (start int, end int, boundedPage int, boundedPerPage int) {
	if page < 1 {
		page = 1
	}
	if page > 1000000 {
		page = 1000000
	}
	if perPage < 1 {
		perPage = 100
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := int64(page-1) * int64(perPage)
	if offset >= int64(total) {
		return total, total, page, perPage
	}

	start = int(offset)
	end64 := offset + int64(perPage)
	if end64 > int64(total) {
		end = total
	} else {
		end = int(end64)
	}

	return start, end, page, perPage
}

func getLocalStorage(c *gin.Context, reqPath string) (*local.Local, string, error) {
	user := c.MustGet("user").(*model.User)
	fullPath, err := user.JoinPath(reqPath)
	if err != nil {
		return nil, "", err
	}
	if !common.CanReadPathByRole(user, fullPath) {
		return nil, "", errors.New("permission denied")
	}

	storage, actualPath, err := op.GetStorageAndActualPath(fullPath)
	if err != nil {
		return nil, "", err
	}
	storageLocal, ok := storage.(*local.Local)
	if !ok || storageLocal.State() == nil {
		return nil, "", errors.New("operation only supported on local storage")
	}
	return storageLocal, actualPath, nil
}

// Trash Handlers

type TrashListReq struct {
	Path    string `form:"path"`
	Page    int    `form:"page"`
	PerPage int    `form:"per_page"`
}

type TrashItemResp struct {
	TrashID      string `json:"trash_id"`
	OriginalPath string `json:"original_path"`
	Size         int64  `json:"size"`
	IsDir        bool   `json:"is_dir"`
	DeletedAt    string `json:"deleted_at"`
}

func FsTrashList(c *gin.Context) {
	reqCtx := c.Request.Context()
	var req TrashListReq
	_ = c.ShouldBindQuery(&req)
	if req.Path == "" {
		req.Path = "/"
	}

	storageLocal, _, err := getLocalStorage(c, req.Path)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	items, err := storageLocal.ListTrash(reqCtx)
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}

	total := len(items)
	start, end, page, perPage := parsePaginationSlice(req.Page, req.PerPage, total)

	resp := make([]TrashItemResp, 0, end-start)
	for _, item := range items[start:end] {
		resp = append(resp, TrashItemResp{
			TrashID:      item.TrashID,
			OriginalPath: item.OriginalPath,
			Size:         item.Size,
			IsDir:        item.IsDir,
			DeletedAt:    item.DeletedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	common.SuccessResp(c, gin.H{
		"content":  resp,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

type TrashRestoreReq struct {
	Path       string `json:"path" binding:"required"`
	TrashID    string `json:"trash_id" binding:"required"`
	TargetPath string `json:"target_path"`
}

func FsTrashRestore(c *gin.Context) {
	reqCtx := c.Request.Context()
	var req TrashRestoreReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	storageLocal, _, err := getLocalStorage(c, req.Path)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	// Cross-storage validation
	trashItem, err := storageLocal.State().GetTrashItem(reqCtx, req.TrashID)
	if err != nil || trashItem.StorageID != storageLocal.GetStorage().ID {
		common.ErrorStrResp(c, "trash item not found", 404)
		return
	}

	rec, err := storageLocal.RestoreTrash(reqCtx, req.TrashID, req.TargetPath)
	if err != nil {
		if errors.Is(err, localstate.ErrDestinationExists) {
			common.ErrorStrResp(c, "destination already exists", 409)
			return
		}
		common.ErrorResp(c, err, 500)
		return
	}

	// Clear op cache
	if rec != nil {
		parentDir := stdpath.Dir(rec.RelativePath)
		op.ClearCache(storageLocal, parentDir)
	}

	common.SuccessResp(c)
}

type TrashDeleteReq struct {
	Path    string `json:"path" binding:"required"`
	TrashID string `json:"trash_id" binding:"required"`
	Confirm bool   `json:"confirm"`
}

func FsTrashDelete(c *gin.Context) {
	reqCtx := c.Request.Context()
	var req TrashDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if !req.Confirm {
		common.ErrorStrResp(c, "confirm is required to permanently delete trash", 400)
		return
	}

	storageLocal, _, err := getLocalStorage(c, req.Path)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	// Cross-storage validation
	trashItem, err := storageLocal.State().GetTrashItem(reqCtx, req.TrashID)
	if err != nil || trashItem.StorageID != storageLocal.GetStorage().ID {
		common.ErrorStrResp(c, "trash item not found", 404)
		return
	}

	if err := storageLocal.PurgeTrash(reqCtx, req.TrashID); err != nil {
		common.ErrorResp(c, err, 500)
		return
	}

	common.SuccessResp(c)
}

// Version Handlers

type VersionsReq struct {
	Path    string `form:"path" binding:"required"`
	Page    int    `form:"page"`
	PerPage int    `form:"per_page"`
}

type VersionItemResp struct {
	VersionID string `json:"version_id"`
	Revision  int64  `json:"revision"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

func FsVersionsList(c *gin.Context) {
	reqCtx := c.Request.Context()
	var req VersionsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	storageLocal, actualPath, err := getLocalStorage(c, req.Path)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	rec, err := storageLocal.GetRecord(reqCtx, actualPath)
	if err != nil {
		common.ErrorStrResp(c, "file not found", 404)
		return
	}

	versions, err := storageLocal.ListVersions(reqCtx, actualPath)
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}

	total := len(versions)
	start, end, page, perPage := parsePaginationSlice(req.Page, req.PerPage, total)

	resp := make([]VersionItemResp, 0, end-start)
	for _, v := range versions[start:end] {
		resp = append(resp, VersionItemResp{
			VersionID: v.VersionID,
			Revision:  v.Revision,
			Size:      v.Size,
			CreatedAt: v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	common.SuccessResp(c, gin.H{
		"file_id":  rec.FileID,
		"revision": rec.Revision,
		"content":  resp,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

type VersionRestoreReq struct {
	Path      string `json:"path" binding:"required"`
	VersionID string `json:"version_id" binding:"required"`
	Revision  int64  `json:"revision"`
}

func FsVersionsRestore(c *gin.Context) {
	reqCtx := c.Request.Context()
	var req VersionRestoreReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if req.Revision <= 0 {
		common.ErrorStrResp(c, "positive revision is required for version restore", 400)
		return
	}

	storageLocal, actualPath, err := getLocalStorage(c, req.Path)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	rec, err := storageLocal.GetRecord(reqCtx, actualPath)
	if err != nil {
		common.ErrorStrResp(c, "file not found", 404)
		return
	}

	// Cross-file / cross-storage check
	fv, err := storageLocal.State().GetFileVersion(reqCtx, req.VersionID)
	if err != nil || fv.StorageID != storageLocal.GetStorage().ID || fv.FileID != rec.FileID {
		common.ErrorStrResp(c, "version does not belong to specified file", 404)
		return
	}

	updatedRec, err := storageLocal.RestoreVersion(reqCtx, rec.FileID, req.VersionID, req.Revision)
	if err != nil {
		if errors.Is(err, localstate.ErrConflict) {
			common.ErrorStrResp(c, "revision conflict", 409)
			return
		}
		common.ErrorResp(c, err, 500)
		return
	}

	parentDir := stdpath.Dir(actualPath)
	op.ClearCache(storageLocal, parentDir)

	common.SuccessResp(c, gin.H{
		"revision": updatedRec.Revision,
	})
}

// Text Handlers

type TextGetReq struct {
	Path string `form:"path" binding:"required"`
}

func FsTextGet(c *gin.Context) {
	reqCtx := c.Request.Context()
	var req TextGetReq
	if err := c.ShouldBindQuery(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	if !localstate.IsAllowedTextExtension(req.Path) {
		common.ErrorStrResp(c, "file extension not in text editor allowlist", 400)
		return
	}

	storageLocal, actualPath, err := getLocalStorage(c, req.Path)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	content, revision, err := storageLocal.State().ReadText(reqCtx, actualPath)
	if err != nil {
		if errors.Is(err, localstate.ErrNotFound) {
			common.ErrorStrResp(c, "file not found", 404)
			return
		}
		if errors.Is(err, localstate.ErrTextTooLarge) ||
			errors.Is(err, localstate.ErrInvalidTextExtension) ||
			errors.Is(err, localstate.ErrInvalidUTF8) ||
			errors.Is(err, localstate.ErrNulByte) {
			common.ErrorStrResp(c, err.Error(), 400)
			return
		}
		if errors.Is(err, localstate.ErrSymlinkForbidden) {
			common.ErrorResp(c, err, 403)
			return
		}
		common.ErrorResp(c, err, 500)
		return
	}

	common.SuccessResp(c, gin.H{
		"content":  content,
		"revision": revision,
	})
}

type TextPutReq struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Revision int64  `json:"revision"`
}

func FsTextPut(c *gin.Context) {
	reqCtx := c.Request.Context()
	// Worst-case JSON encoding overhead for 2MiB UTF-8 text + JSON keys/paths
	maxBody := int64(6*localstate.MaxTextFileSize + 65536)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBody)

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	// Reject raw invalid UTF-8 before JSON decoder replaces bytes with replacement characters
	if !utf8.Valid(rawBody) {
		common.ErrorStrResp(c, "request body contains invalid UTF-8", 400)
		return
	}

	var req TextPutReq
	if err := json.Unmarshal(rawBody, &req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if req.Path == "" {
		common.ErrorStrResp(c, "path is required", 400)
		return
	}

	if !localstate.IsAllowedTextExtension(req.Path) {
		common.ErrorStrResp(c, "file extension not in text editor allowlist", 400)
		return
	}

	data := []byte(req.Content)
	if err := localstate.ValidateTextContent(data); err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}

	storageLocal, actualPath, err := getLocalStorage(c, req.Path)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}

	// For an existing file, require positive revision (UI edits existing only)
	normRel, err := storageLocal.State().CanonicalRelPath(actualPath)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if fi, statErr := storageLocal.State().OsRoot().Stat(normRel); statErr == nil && !fi.IsDir() {
		if req.Revision <= 0 {
			common.ErrorStrResp(c, "positive revision is required for existing file edit", 400)
			return
		}
	}

	newRevision, err := storageLocal.State().SaveText(reqCtx, actualPath, data, req.Revision)
	if err != nil {
		if errors.Is(err, localstate.ErrConflict) {
			common.ErrorStrResp(c, "revision conflict", 409)
			return
		}
		if errors.Is(err, localstate.ErrTextTooLarge) ||
			errors.Is(err, localstate.ErrInvalidTextExtension) ||
			errors.Is(err, localstate.ErrInvalidUTF8) ||
			errors.Is(err, localstate.ErrNulByte) {
			common.ErrorStrResp(c, err.Error(), 400)
			return
		}
		common.ErrorResp(c, err, 500)
		return
	}

	parentDir := stdpath.Dir(actualPath)
	op.ClearCache(storageLocal, parentDir)

	common.SuccessResp(c, gin.H{
		"revision": newRevision,
	})
}
