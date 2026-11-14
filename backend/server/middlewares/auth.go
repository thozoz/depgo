package middlewares

import (
	"crypto/subtle"
	"errors"
	"fmt"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/device"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Auth is a middleware that checks if the user is logged in.
// if token is empty, set user to guest
func Auth(c *gin.Context) {
	token := c.GetHeader("Authorization")
	adminToken := setting.GetStr(conf.Token)
	if adminToken != "" && token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(adminToken)) == 1 {
		admin, err := op.GetAdmin()
		if err != nil {
			common.ErrorResp(c, err, 500)
			c.Abort()
			return
		}
		if !HandleSession(c, admin) {
			return
		}
		log.Debugf("use admin token: %+v", admin)
		c.Next()
		return
	}
	if token == "" {
		guest, err := op.GetGuest()
		if err != nil {
			common.ErrorResp(c, err, 500)
			c.Abort()
			return
		}
		if guest.Disabled {
			common.ErrorStrResp(c, "Guest user is disabled, login please", 401)
			c.Abort()
			return
		}
		if len(guest.Role) > 0 {
			roles, err := op.GetRolesByUserID(guest.ID)
			if err != nil {
				common.ErrorStrResp(c, fmt.Sprintf("Fail to load guest roles: %v", err), 500)
				c.Abort()
				return
			}
			guest.RolesDetail = roles
		}
		if !HandleSession(c, guest) {
			return
		}
		log.Debugf("use empty token: %+v", guest)
		c.Next()
		return
	}
	userClaims, err := common.ParseToken(token)
	if err != nil {
		common.ErrorResp(c, err, 401)
		c.Abort()
		return
	}
	user, err := op.GetUserByName(userClaims.Username)
	if err != nil {
		common.ErrorResp(c, err, 401)
		c.Abort()
		return
	}
	// validate password timestamp
	if userClaims.PwdTS != user.PwdTS {
		common.ErrorStrResp(c, "Password has been changed, login please", 401)
		c.Abort()
		return
	}
	if user.Disabled {
		common.ErrorStrResp(c, "Current user is disabled, replace please", 401)
		c.Abort()
		return
	}
	if len(user.Role) > 0 {
		roles, err := op.GetRolesByUserID(user.ID)
		if err != nil {
			common.ErrorStrResp(c, fmt.Sprintf("Fail to load roles: %v", err), 500)
			c.Abort()
			return
		}
		user.RolesDetail = roles
	}
	if !bindClaimDeviceKey(c, userClaims) || !HandleSession(c, user) {
		return
	}
	log.Debugf("use login token: %+v", user)
	c.Next()
}

// bindClaimDeviceKey records the device session the token was minted for, so
// HandleSession uses it instead of hashing a client-supplied Client-Id.
//
// The key must come from the signed claim: deriving it from the header let a
// bearer of a token whose session had been evicted (or retired by max_devices)
// send a different or absent Client-Id, get a fresh active session created, and
// keep using a token that was supposed to be dead.
//
// A token without the claim predates this binding and is refused rather than
// silently downgraded to the header-derived key. Tokens are only valid while
// their entry is in the in-memory cache, so every one of them is reissued on
// restart anyway.
func bindClaimDeviceKey(c *gin.Context, claims *common.UserClaims) bool {
	if claims.DeviceKey == "" {
		common.ErrorStrResp(c, "token is not bound to a device session, login please", 401)
		c.Abort()
		return false
	}
	c.Set("device_key_from_claims", claims.DeviceKey)
	return true
}

// HandleSession verifies device sessions and stores context values.
func HandleSession(c *gin.Context, user *model.User) bool {
	key := sessionKey(c, user)
	handle := device.Handle
	if c.GetString("device_key_from_claims") != "" {
		handle = device.HandleBound
	}
	if err := handle(user.ID, key, c.Request.UserAgent(), c.ClientIP()); err != nil {
		token := c.GetHeader("Authorization")
		if errors.Is(err, errs.SessionInactive) {
			_ = common.InvalidateToken(token)
			common.ErrorResp(c, err, 401)
		} else {
			common.ErrorResp(c, err, 403)
		}
		c.Abort()
		return false
	}
	c.Set("device_key", key)
	c.Set("user", user)
	return true
}

func Authn(c *gin.Context) {
	token := c.GetHeader("Authorization")
	adminToken := setting.GetStr(conf.Token)
	if adminToken != "" && token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(adminToken)) == 1 {
		admin, err := op.GetAdmin()
		if err != nil {
			common.ErrorResp(c, err, 500)
			c.Abort()
			return
		}
		c.Set("user", admin)
		log.Debugf("use admin token: %+v", admin)
		c.Next()
		return
	}
	if token == "" {
		guest, err := op.GetGuest()
		if err != nil {
			common.ErrorResp(c, err, 500)
			c.Abort()
			return
		}
		c.Set("user", guest)
		log.Debugf("use empty token: %+v", guest)
		c.Next()
		return
	}
	userClaims, err := common.ParseToken(token)
	if err != nil {
		common.ErrorResp(c, err, 401)
		c.Abort()
		return
	}
	user, err := op.GetUserByName(userClaims.Username)
	if err != nil {
		common.ErrorResp(c, err, 401)
		c.Abort()
		return
	}
	// validate password timestamp
	if userClaims.PwdTS != user.PwdTS {
		common.ErrorStrResp(c, "Password has been changed, login please", 401)
		c.Abort()
		return
	}
	if user.Disabled {
		common.ErrorStrResp(c, "Current user is disabled, replace please", 401)
		c.Abort()
		return
	}
	if len(user.Role) > 0 {
		var roles []model.Role
		for _, roleID := range user.Role {
			role, err := op.GetRole(uint(roleID))
			if err != nil {
				common.ErrorStrResp(c, fmt.Sprintf("load role %d failed", roleID), 500)
				c.Abort()
				return
			}
			roles = append(roles, *role)
		}
		user.RolesDetail = roles
	}
	// Authn does not upsert sessions, but it must still refuse a token whose
	// device session was deactivated; otherwise a route guarded by it accepts an
	// evicted credential.
	if !bindClaimDeviceKey(c, userClaims) {
		return
	}
	if err := device.CheckActive(user.ID, userClaims.DeviceKey); err != nil {
		_ = common.InvalidateToken(token)
		common.ErrorResp(c, err, 401)
		c.Abort()
		return
	}
	c.Set("device_key", userClaims.DeviceKey)
	c.Set("user", user)
	log.Debugf("use login token: %+v", user)
	c.Next()
}

// sessionKey returns the device session key for a request.
//
// A token minted by login carries the key it was bound to, and that claim wins:
// the Client-Id header is client input, so trusting it let a token whose session
// had been deactivated resurrect itself under a different key. Header derivation
// remains only for credentials that have no session claim at all (the site admin
// token and the anonymous guest).
func sessionKey(c *gin.Context, user *model.User) string {
	if keyVal, ok := c.Get("device_key_from_claims"); ok {
		if key, ok := keyVal.(string); ok && key != "" {
			return key
		}
	}
	clientID := c.GetHeader("Client-Id")
	if clientID == "" {
		clientID = c.Query("client_id")
	}
	return utils.GetMD5EncodeStr(fmt.Sprintf("%d-%s", user.ID, clientID))
}

func AuthNotGuest(c *gin.Context) {
	user := c.MustGet("user").(*model.User)
	if user.IsGuest() {
		common.ErrorStrResp(c, "You are a guest", 403)
		c.Abort()
	} else {
		c.Next()
	}
}

func AuthAdmin(c *gin.Context) {
	user := c.MustGet("user").(*model.User)
	if !user.IsAdmin() {
		common.ErrorStrResp(c, "You are not an admin", 403)
		c.Abort()
	} else {
		c.Next()
	}
}
