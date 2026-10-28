package handles

import (
	stdpath "path"
	"time"

	"github.com/alist-org/alist/v3/internal/db"
	shareauth "github.com/alist-org/alist/v3/internal/share"

	"github.com/alist-org/alist/v3/internal/fs"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// bindPublicShareReq keeps struct, field and validator details out of the
// public response; they stay in the server log.
func bindPublicShareReq(c *gin.Context, req any) bool {
	if err := c.ShouldBind(req); err != nil {
		log.Warnf("public share request rejected: %v", err)
		common.ErrorStrResp(c, "invalid request", 400)
		return false
	}
	return true
}

func GetPublicShareInfo(c *gin.Context) {
	shareID := c.Query("share_id")
	if shareID == "" {
		common.ErrorStrResp(c, "share_id is required", 400)
		return
	}
	share, err := db.GetShareByShareID(shareID)
	if err != nil {
		common.ErrorResp(c, err, 404)
		return
	}
	if !ensureShareAvailable(c, share) {
		return
	}
	token := getShareAccessToken(c, "")
	authed := !share.HasPassword()
	if share.HasPassword() && token != "" {
		authed = shareauth.VerifyAccess(share, token) == nil
	}
	if authed {
		_ = db.TouchShareView(share.ShareID)
	}
	resp := PublicShareInfoResp{
		ShareID:       share.ShareID,
		IsDir:         share.IsDir,
		HasPassword:   share.HasPassword(),
		BurnAfterRead: share.EffectiveAccessLimit() == 1,
		AccessLimit:   share.EffectiveAccessLimit(),
		AllowPreview:  share.AllowPreview,
		AllowDownload: share.AllowDownload,
		Authed:        authed,
		ExpiresAt:     share.ExpiresAt,
	}
	// A password-protected share discloses its name and usage metadata only to
	// callers that already proved knowledge of the password.
	if authed {
		resp.Name = share.Name
		resp.AccessCount = share.AccessCount
		resp.RemainingAccesses = share.RemainingAccesses()
		resp.ConsumedAt = share.ConsumedAt
		resp.CreatedAt = share.CreatedAt
	}
	common.SuccessResp(c, resp)
}

func AuthPublicShare(c *gin.Context) {
	var req ShareAuthReq
	if !bindPublicShareReq(c, &req) {
		return
	}
	share, err := db.GetShareByShareID(req.ShareID)
	if err != nil {
		common.ErrorResp(c, err, 404)
		return
	}
	if !ensureShareAvailable(c, share) {
		return
	}
	if !sharePasswordMatched(share, req.Password) {
		common.ErrorStrResp(c, "password is incorrect", 403)
		return
	}
	token := ""
	if share.HasPassword() {
		ttl := shareAccessTokenLifetime
		if share.ExpiresAt != nil {
			remaining := time.Until(*share.ExpiresAt)
			if remaining <= 0 {
				common.ErrorStrResp(c, "share is expired", 410)
				return
			}
			if remaining < ttl {
				ttl = remaining
			}
		}
		token = shareauth.SignAccess(share, ttl)
	}
	_ = db.TouchShareView(share.ShareID)
	common.SuccessResp(c, gin.H{"token": token})
}

func ListPublicShare(c *gin.Context) {
	var req PublicShareListReq
	if !bindPublicShareReq(c, &req) {
		return
	}
	req.Page, req.PerPage = normalizeListPage(req.Page, req.PerPage)
	share, err := db.GetShareByShareID(req.ShareID)
	if err != nil {
		common.ErrorResp(c, err, 404)
		return
	}
	if !ensureShareAvailable(c, share) {
		return
	}
	token := getShareAccessToken(c, req.Token)
	if !ensureShareAccess(c, share, token) {
		return
	}
	targetPath, relPath, err := resolveShareTarget(share, req.Path)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	obj, err := fs.Get(c, targetPath, &fs.GetArgs{})
	if err != nil {
		common.ErrorResp(c, err, 404)
		return
	}
	if !obj.IsDir() {
		common.ErrorStrResp(c, "path is not a directory", 400)
		return
	}
	objs, err := fs.List(c, targetPath, &fs.ListArgs{})
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	total, pageObjs := pagination(objs, &req.PageReq)
	content := make([]PublicShareObjResp, 0, len(pageObjs))
	for _, item := range pageObjs {
		itemRelPath := relPath
		if itemRelPath == "/" {
			itemRelPath = stdpath.Join("/", item.GetName())
		} else {
			itemRelPath = stdpath.Join(relPath, item.GetName())
		}
		itemTargetPath, _, err := resolveShareTarget(share, itemRelPath)
		if err != nil {
			continue
		}
		content = append(content, toPublicShareObjResp(c, share, item, itemTargetPath, itemRelPath, token))
	}
	common.SuccessResp(c, PublicShareListResp{
		Content:    content,
		Total:      int64(total),
		Page:       req.Page,
		PerPage:    req.PerPage,
		HasMore:    req.PerPage != AllPerPage && req.Page*req.PerPage < total,
		PagesTotal: calcPagesTotal(total, req.PerPage),
	})
}

func GetPublicShare(c *gin.Context) {
	var req PublicShareReq
	if !bindPublicShareReq(c, &req) {
		return
	}
	share, err := db.GetShareByShareID(req.ShareID)
	if err != nil {
		common.ErrorResp(c, err, 404)
		return
	}
	if !ensureShareAvailable(c, share) {
		return
	}
	token := getShareAccessToken(c, req.Token)
	if !ensureShareAccess(c, share, token) {
		return
	}
	targetPath, relPath, err := resolveShareTarget(share, req.Path)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	obj, err := fs.Get(c, targetPath, &fs.GetArgs{})
	if err != nil {
		common.ErrorResp(c, err, 404)
		return
	}
	provider := "unknown"
	storage, storageErr := fs.GetStorage(targetPath, &fs.GetStoragesArgs{})
	if storageErr == nil {
		provider = storage.GetStorage().Driver
	}
	common.SuccessResp(c, PublicShareGetResp{
		Item:     toPublicShareObjResp(c, share, obj, targetPath, relPath, token),
		Provider: provider,
	})
}

func ShareDown(c *gin.Context) {
	share, err := db.GetShareByShareID(c.Param("share_id"))
	if err != nil {
		common.ErrorResp(c, err, 404)
		return
	}
	if !ensureShareAvailable(c, share) {
		return
	}
	if !share.AllowDownload {
		common.ErrorStrResp(c, "download is not allowed", 403)
		return
	}
	token := getShareAccessToken(c, "")
	if !ensureShareAccess(c, share, token) {
		return
	}
	targetPath, _, err := resolveShareWildcardTarget(share, c.Param("path"))
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	obj, err := fs.Get(c, targetPath, &fs.GetArgs{})
	if err != nil {
		common.ErrorResp(c, err, 404)
		return
	}
	if obj.IsDir() {
		common.ErrorStrResp(c, "directory download is not supported", 400)
		return
	}
	if shouldTrackShareContentAccess(c) {
		_ = db.TouchShareDownload(share.ShareID)
		if err := recordShareAccess(share); err != nil {
			common.ErrorResp(c, err, 500, true)
			return
		}
	}
	c.Set("path", targetPath)
	Down(c)
}

func ShareProxy(c *gin.Context) {
	share, err := db.GetShareByShareID(c.Param("share_id"))
	if err != nil {
		common.ErrorResp(c, err, 404)
		return
	}
	if !ensureShareAvailable(c, share) {
		return
	}
	if !share.AllowPreview {
		common.ErrorStrResp(c, "preview is not allowed", 403)
		return
	}
	token := getShareAccessToken(c, "")
	if !ensureShareAccess(c, share, token) {
		return
	}
	targetPath, _, err := resolveShareWildcardTarget(share, c.Param("path"))
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	obj, err := fs.Get(c, targetPath, &fs.GetArgs{})
	if err != nil {
		common.ErrorResp(c, err, 404)
		return
	}
	if obj.IsDir() {
		common.ErrorStrResp(c, "directory preview is not supported", 400)
		return
	}
	if shouldTrackShareContentAccess(c) {
		if err := recordShareAccess(share); err != nil {
			common.ErrorResp(c, err, 500, true)
			return
		}
	}
	c.Set("path", targetPath)
	Proxy(c)
}
