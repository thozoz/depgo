package middlewares

import (
	"crypto/subtle"
	"strings"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/setting"

	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func Down(verifyFunc func(string, string) error) func(c *gin.Context) {
	return func(c *gin.Context) {
		rawPath := parsePath(c.Param("path"))
		c.Set("path", rawPath)
		meta, err := op.GetNearestMeta(rawPath)
		if err != nil {
			if !errors.Is(errors.Cause(err), errs.MetaNotFound) {
				common.ErrorResp(c, err, 500, true)
				return
			}
		}
		c.Set("meta", meta)
		// verify sign or auth token
		s := c.Query("sign")
		if s != "" {
			err = verifyFunc(rawPath, strings.TrimSuffix(s, "/"))
			if err != nil {
				common.ErrorResp(c, err, 401)
				c.Abort()
				return
			}
		} else if !authorizeDownloadToken(c, rawPath, meta) {
			return
		}
		c.Next()
	}
}

// authorizeDownloadToken accepts a request that carries no download signature
// only when the Authorization header is a real credential: the configured admin
// token or a session token for an existing, enabled account. A mere non-empty
// header value must never unlock a file.
//
// The session-token path must also prove read permission for rawPath. Without
// that, any logged-in user could fetch any file by naming its path directly:
// /d and /p take the URL path verbatim, so they bypass both the caller's base
// path and their role's permission scopes. The signature branch is exempt because
// a signature is only ever minted by a handler that already performed the
// permission checks.
func authorizeDownloadToken(c *gin.Context, rawPath string, meta *model.Meta) bool {
	token := c.GetHeader("Authorization")
	if token == "" {
		common.ErrorStrResp(c, "download signature or authentication required", 401)
		c.Abort()
		return false
	}
	if adminToken := setting.GetStr(conf.Token); adminToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(adminToken)) == 1 {
		return true
	}
	claims, err := common.ParseToken(token)
	if err != nil {
		common.ErrorStrResp(c, "invalid or expired token", 401)
		c.Abort()
		return false
	}
	user, err := op.GetUserByName(claims.Username)
	if err != nil || user.Disabled || claims.PwdTS != user.PwdTS {
		common.ErrorStrResp(c, "invalid or expired token", 401)
		c.Abort()
		return false
	}
	// Reuse the session check so an evicted or expired device session cannot
	// keep downloading with a still-valid JWT.
	if !HandleSession(c, user) {
		return false
	}
	if !canReadDownloadPath(user, meta, rawPath) {
		// Generic message: whether the path exists or is merely forbidden should
		// not be learnable from the response.
		common.ErrorStrResp(c, "permission denied", 403)
		c.Abort()
		return false
	}
	return true
}

// canReadDownloadPath reports whether user may download rawPath.
//
// It mirrors the checks the listing and metadata handlers perform: the path must
// sit under the user's base path (which encodes their path limit), the role must
// grant read there, and a password-protected meta must not block it. A password
// cannot be supplied on a bare /d request, so a password-protected meta denies
// access here - such content is normally reached through a signed URL or a share.
func canReadDownloadPath(user *model.User, meta *model.Meta, rawPath string) bool {
	if user == nil {
		return false
	}
	// Admins keep unrestricted access; that is what the admin role means.
	if user.IsAdmin() {
		return true
	}
	if !utils.IsSubPath(user.BasePath, rawPath) {
		return false
	}
	if !common.CanReadPathByRole(user, rawPath) {
		return false
	}
	return common.CanAccessWithRoles(user, meta, rawPath, "")
}

func parsePath(path string) string {
	// Gin has already decoded the URL path. Decoding again corrupts literal
	// percent signs and filenames containing escape sequences such as %2F.
	return utils.FixAndCleanPath(path)
}
