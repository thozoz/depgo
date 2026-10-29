package handles

import (
	"bytes"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"path"
	"strings"
	"time"

	"github.com/Xhofe/go-cache"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/device"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/session"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
)

var loginCache = cache.NewMemCache[int]()
var (
	defaultDuration            = time.Minute * 5
	defaultTimes               = 5
	invalidLoginCredentialsMsg = "username or password is incorrect"
)

// recordLoginFailure increments the failure counter for key while giving the
// whole entry a fresh window.
//
// The counter used to be stored without any expiry, so failures accumulated
// forever: four typos today plus one next week locked the caller out, even
// though no burst ever happened. Attaching the window to each write makes the
// limit mean "defaultTimes failures within defaultDuration", which is what the
// message and the lockout length already imply.
func recordLoginFailure(key string) {
	count, _ := loginCache.Get(key)
	loginCache.Set(key, count+1, cache.WithEx[int](defaultDuration))
}

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password"`
	OtpCode  string `json:"otp_code"`
}

// Login Deprecated
func Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	req.Password = model.StaticHash(req.Password)
	loginHash(c, &req)
}

// LoginHash login with password hashed by sha256
func LoginHash(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	loginHash(c, &req)
}

func loginHash(c *gin.Context, req *LoginReq) {
	// check count of login
	ip := c.ClientIP()
	count, ok := loginCache.Get(ip)
	if ok && count >= defaultTimes {
		common.ErrorStrResp(c, "Too many unsuccessful sign-in attempts have been made using an incorrect username or password, Try again later.", 429)
		loginCache.Expire(ip, defaultDuration)
		return
	}
	// check username
	user, err := op.GetUserByName(req.Username)
	if err != nil {
		common.ErrorStrResp(c, invalidLoginCredentialsMsg, 400)
		recordLoginFailure(ip)
		return
	}
	// validate password hash
	if err := user.ValidatePwdStaticHash(req.Password); err != nil {
		common.ErrorStrResp(c, invalidLoginCredentialsMsg, 400)
		recordLoginFailure(ip)
		return
	}
	// check 2FA
	if user.OtpSecret != "" {
		if !totp.Validate(req.OtpCode, user.OtpSecret) {
			common.ErrorStrResp(c, "Invalid 2FA code", 402)
			recordLoginFailure(ip)
			return
		}
	}

	clientID := c.GetHeader("Client-Id")
	if clientID == "" {
		clientID = c.Query("client_id")
	}
	key := utils.GetMD5EncodeStr(fmt.Sprintf("%d-%s",
		user.ID, clientID))

	if err := device.EnsureActiveOnLogin(user.ID, key, c.Request.UserAgent(), c.ClientIP()); err != nil {
		if errors.Is(err, errs.TooManyDevices) {
			common.ErrorResp(c, err, 403)
		} else {
			common.ErrorResp(c, err, 400, true)
		}
		return
	}

	// generate token
	token, err := common.GenerateToken(user)
	if err != nil {
		common.ErrorResp(c, err, 400, true)
		return
	}
	common.SuccessResp(c, gin.H{"token": token, "device_key": key})
	loginCache.Del(ip)
}

type RegisterReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Register a new user
func Register(c *gin.Context) {
	if !setting.GetBool(conf.AllowRegister) {
		common.ErrorStrResp(c, "registration is disabled", 403)
		return
	}
	var req RegisterReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	user := &model.User{
		Username: req.Username,
		Role:     model.Roles{op.GetDefaultRoleID()},
		Authn:    "[]",
	}
	user.SetPassword(req.Password)
	if err := op.CreateUser(user); err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c)
}

type UserResp struct {
	model.User
	Otp         bool                    `json:"otp"`
	RoleNames   []string                `json:"role_names"`
	Permissions []model.PermissionEntry `json:"permissions"`
}

// CurrentUser get current user by token
// if token is empty, return guest user
func CurrentUser(c *gin.Context) {
	user := c.MustGet("user").(*model.User)

	userResp := UserResp{
		User: *user,
	}
	userResp.Password = ""
	if userResp.OtpSecret != "" {
		userResp.Otp = true
	}

	var roleNames []string
	permMap := map[string]int32{}
	paths := make([]string, 0)

	for _, role := range user.RolesDetail {
		roleNames = append(roleNames, role.Name)
		for _, entry := range role.PermissionScopes {
			cleanPath := path.Clean("/" + strings.TrimPrefix(entry.Path, "/"))
			if _, ok := permMap[cleanPath]; !ok {
				paths = append(paths, cleanPath)
			}
			permMap[cleanPath] |= entry.Permission
		}
	}
	userResp.RoleNames = roleNames

	for _, fullPath := range paths {
		userResp.Permissions = append(userResp.Permissions, model.PermissionEntry{
			Path:       fullPath,
			Permission: permMap[fullPath],
		})
	}

	common.SuccessResp(c, userResp)
}

// UpdateCurrentReq is the self-service profile update payload.
//
// Every field is optional, so a client can change the password without resending
// the username (and vice versa). Requiring username made a password-only change
// impossible, which broke the ordinary "change my password" flow.
//
// CurrentPassword is required whenever Password is set (except when the caller
// authenticated with the site admin token). Without it, anyone holding a stolen
// session token could set a new password and lock the real owner out of their own
// account; asking for the current password means token theft alone is not enough
// to take ownership.
//
// sso_id is deliberately absent. It used to be accepted here, which meant any
// logged-in user could write an arbitrary value into their own SsoID column.
// SSO login resolves the account with db.GetUserBySSOID and then issues a token
// for whatever user that returns, so a user could set their SsoID to another
// user's identifier and, on their next SSO sign-in, receive that user's
// session. The binding belongs to the verified SSO flow, never to a profile
// form.
type UpdateCurrentReq struct {
	Username        *string `json:"username"`
	Password        *string `json:"password"`
	CurrentPassword *string `json:"current_password"`
}

func UpdateCurrent(c *gin.Context) {
	var req UpdateCurrentReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	user := c.MustGet("user").(*model.User)
	if user.IsGuest() {
		common.ErrorStrResp(c, "Guest user can not update profile", 403)
		return
	}
	if req.Username == nil && req.Password == nil {
		common.ErrorStrResp(c, "nothing to update", 400)
		return
	}

	// Work on a copy: the value in the request context is the cached user and
	// mutating it before the update succeeds would leave the cache describing
	// state that was never persisted.
	updated := *user
	updated.RolesDetail = nil
	updated.Password = ""
	updated.SsoID = user.SsoID // not settable here; keep what is stored

	if req.Username != nil {
		name := strings.TrimSpace(*req.Username)
		if name == "" {
			common.ErrorStrResp(c, "username can not be empty", 400)
			return
		}
		if name != user.Username {
			if other, err := op.GetUserByName(name); err == nil && other.ID != user.ID {
				common.ErrorStrResp(c, "username is already taken", 400)
				return
			}
			updated.Username = name
		}
	}
	if req.Password != nil {
		if *req.Password == "" {
			common.ErrorStrResp(c, "password can not be empty", 400)
			return
		}
		// A stolen session token must not be enough to take ownership of the
		// account: prove knowledge of the current password first. The site admin
		// token is exempt because it already represents full control and has its
		// own administrative reset path.
		if !isSiteAdminToken(c) {
			if req.CurrentPassword == nil || *req.CurrentPassword == "" {
				common.ErrorStrResp(c, "current password is required to set a new one", 400)
				return
			}
			if err := updated.ValidateRawPassword(*req.CurrentPassword); err != nil {
				common.ErrorStrResp(c, "current password is incorrect", 403)
				return
			}
		}
		updated.SetPassword(*req.Password)
	}

	if err := op.UpdateUser(&updated); err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c)
}

func Generate2FA(c *gin.Context) {
	user := c.MustGet("user").(*model.User)
	if user.IsGuest() {
		common.ErrorStrResp(c, "Guest user can not generate 2FA code", 403)
		return
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Alist",
		AccountName: user.Username,
	})
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	img, err := key.Image(400, 400)
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	// to base64
	var buf bytes.Buffer
	png.Encode(&buf, img)
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	common.SuccessResp(c, gin.H{
		"qr":     "data:image/png;base64," + b64,
		"secret": key.Secret(),
	})
}

type Verify2FAReq struct {
	Code   string `json:"code" binding:"required"`
	Secret string `json:"secret" binding:"required"`
}

func Verify2FA(c *gin.Context) {
	var req Verify2FAReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	user := c.MustGet("user").(*model.User)
	if user.IsGuest() {
		common.ErrorStrResp(c, "Guest user can not generate 2FA code", 403)
		return
	}
	if !totp.Validate(req.Code, req.Secret) {
		common.ErrorStrResp(c, "Invalid 2FA code", 400)
		return
	}
	user.OtpSecret = req.Secret
	if err := op.UpdateUser(user); err != nil {
		common.ErrorResp(c, err, 500)
	} else {
		common.SuccessResp(c)
	}
}

func LogOut(c *gin.Context) {
	if keyVal, ok := c.Get("device_key"); ok {
		if err := session.MarkInactive(keyVal.(string)); err != nil {
			common.ErrorResp(c, err, 500)
			return
		}
		c.Set("session_inactive", true)
	}
	err := common.InvalidateToken(c.GetHeader("Authorization"))
	if err != nil {
		common.ErrorResp(c, err, 500)
	} else {
		common.SuccessResp(c)
	}
}

// isSiteAdminToken reports whether the request authenticated with the site admin
// token rather than a per-user session.
func isSiteAdminToken(c *gin.Context) bool {
	token := c.GetHeader("Authorization")
	adminToken := setting.GetStr(conf.Token)
	if token == "" || adminToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(adminToken)) == 1
}
