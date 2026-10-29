package handles

import (
	"strconv"

	"github.com/alist-org/alist/v3/pkg/utils"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func ListUsers(c *gin.Context) {
	var req model.PageReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	req.Validate()
	log.Debugf("%+v", req)
	users, total, err := op.GetUsers(req.Page, req.PerPage)
	if err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c, common.PageResp{
		Content: users,
		Total:   total,
	})
}

func CreateUser(c *gin.Context) {
	var req model.User
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if len(req.Role) == 0 {
		req.Role = model.Roles{op.GetDefaultRoleID()}
	}
	if req.IsAdmin() || req.IsGuest() {
		common.ErrorStrResp(c, "admin or guest user can not be created", 400, true)
		return
	}
	req.SetPassword(req.Password)
	req.Password = ""
	req.Authn = "[]"
	if err := op.CreateUser(&req); err != nil {
		common.ErrorResp(c, err, 500, true)
	} else {
		common.SuccessResp(c)
	}
}

// UpdateUserReq is a partial user update.
//
// Every optional field is a pointer so that "not sent" can be told apart from
// "sent as zero". Binding straight into model.User used to make that
// impossible: a request carrying only id + password left Role and Permission
// at their zero values, and the handler then saved the whole struct, silently
// stripping the user's roles and permissions.
type UpdateUserReq struct {
	ID         uint         `json:"id" binding:"required"`
	Username   *string      `json:"username"`
	Password   *string      `json:"password"`
	Role       *model.Roles `json:"role"`
	Permission *int32       `json:"permission"`
	Disabled   *bool        `json:"disabled"`
	// OtpSecret is accepted so the 2FA reset flow keeps working, but an empty
	// value means "leave it alone" rather than "clear it".
	OtpSecret *string `json:"otp_secret"`
}

func UpdateUser(c *gin.Context) {
	var req UpdateUserReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	stored, err := op.GetUserById(req.ID)
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}

	// Work on a copy of the stored user so every field the client did not send
	// keeps its current value.
	updated := *stored
	updated.Password = ""
	updated.RolesDetail = nil

	if req.Username != nil && *req.Username != "" && *req.Username != stored.Username {
		if stored.IsAdmin() {
			common.ErrorStrResp(c, "cannot change username of admin user", 403)
			return
		}
		updated.Username = *req.Username
	}

	if req.Role != nil {
		if !utils.SliceEqual(stored.Role, *req.Role) {
			if stored.IsAdmin() {
				common.ErrorStrResp(c, "cannot change role of admin user", 403)
				return
			}
			if req.Role.Contains(model.ADMIN) || req.Role.Contains(model.GUEST) {
				common.ErrorStrResp(c, "cannot assign admin or guest role to user", 400, true)
				return
			}
			updated.Role = *req.Role
		}
	}

	if req.Permission != nil {
		updated.Permission = *req.Permission
	}

	if req.Password != nil && *req.Password != "" {
		updated.SetPassword(*req.Password)
	}

	if req.OtpSecret != nil && *req.OtpSecret != "" {
		updated.OtpSecret = *req.OtpSecret
	}

	if req.Disabled != nil && *req.Disabled != stored.Disabled {
		if *req.Disabled && stored.IsAdmin() {
			count, err := op.CountEnabledAdminsExcluding(stored.ID)
			if err != nil {
				common.ErrorResp(c, err, 500)
				return
			}
			if count == 0 {
				common.ErrorStrResp(c, "at least one enabled admin must be kept", 400)
				return
			}
		}
		updated.Disabled = *req.Disabled
	}

	if err := op.UpdateUser(&updated); err != nil {
		common.ErrorResp(c, err, 500)
	} else {
		common.SuccessResp(c)
	}
}

func DeleteUser(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if err := op.DeleteUserById(uint(id)); err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c)
}

func GetUser(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	user, err := op.GetUserById(uint(id))
	if err != nil {
		common.GetErrResp(c, err, true)
		return
	}
	common.SuccessResp(c, user)
}

func Cancel2FAById(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if err := op.Cancel2FAById(uint(id)); err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c)
}

func DelUserCache(c *gin.Context) {
	username := c.Query("username")
	err := op.DelUserCache(username)
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c)
}
