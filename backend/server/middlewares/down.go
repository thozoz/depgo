package middlewares

import (
	"crypto/subtle"
	"strings"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/setting"

	"github.com/alist-org/alist/v3/internal/errs"
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
		} else if !authorizeDownloadToken(c) {
			return
		}
		c.Next()
	}
}

// authorizeDownloadToken accepts a request that carries no download signature
// only when the Authorization header is a real credential: the configured admin
// token or a session token for an existing, enabled account. A mere non-empty
// header value must never unlock a file.
func authorizeDownloadToken(c *gin.Context) bool {
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
	return true
}

func parsePath(path string) string {
	// Gin has already decoded the URL path. Decoding again corrupts literal
	// percent signs and filenames containing escape sequences such as %2F.
	return utils.FixAndCleanPath(path)
}
