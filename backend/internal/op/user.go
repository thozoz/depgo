package op

import (
	"github.com/Xhofe/go-cache"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/singleflight"
	"github.com/alist-org/alist/v3/pkg/utils"
)

var userCache = cache.NewMemCache(cache.WithShards[*model.User](2))
var userG singleflight.Group[*model.User]
var guestUser *model.User

func enforceAdminUserDefaults(u *model.User) error {
	if u == nil || !u.IsAdmin() {
		return nil
	}
	changed := false
	if utils.FixAndCleanPath(u.BasePath) != "/" {
		u.BasePath = "/"
		changed = true
	}
	if u.Permission != 0xFFFF {
		u.Permission = 0xFFFF
		changed = true
	}
	if !changed {
		return nil
	}
	return db.UpdateUser(u)
}

// GetAdmin reads the administrator from the database on every call.
//
// It used to be cached for the life of the process. A password reset made with
// the CLI runs in a separate process and can only reach the server's memory over
// HTTP, so a running server kept authenticating the old password (and accepting
// the old tokens) until it restarted. Depgo has one administrator and a local
// SQLite file, so a point query per call is cheaper than that inconsistency.
func GetAdmin() (*model.User, error) {
	role, err := GetRoleByName("admin")
	if err != nil {
		return nil, err
	}
	user, err := db.GetUserByRole(int(role.ID))
	if err != nil {
		return nil, err
	}
	if err := enforceAdminUserDefaults(user); err != nil {
		return nil, err
	}
	return user, nil
}

func GetGuest() (*model.User, error) {
	if guestUser == nil {
		role, err := GetRoleByName("guest")
		if err != nil {
			return nil, err
		}
		user, err := db.GetUserByRole(int(role.ID))
		if err != nil {
			return nil, err
		}
		guestUser = user
	}
	return guestUser, nil
}

func GetUserByRole(role int) (*model.User, error) {
	return db.GetUserByRole(role)
}

func GetUsersByRole(role int) ([]model.User, error) {
	return db.GetUsersByRole(role)
}

// GetUserByName reads the user from the database; see GetAdmin for why there is
// no cache here.
func GetUserByName(username string) (*model.User, error) {
	if username == "" {
		return nil, errs.EmptyUsername
	}
	user, err, _ := userG.Do(username, func() (*model.User, error) {
		_user, err := db.GetUserByName(username)
		if err != nil {
			return nil, err
		}
		if err := enforceAdminUserDefaults(_user); err != nil {
			return nil, err
		}
		return _user, nil
	})
	return user, err
}

func GetUserById(id uint) (*model.User, error) {
	return db.GetUserById(id)
}

func GetUsers(pageIndex, pageSize int) (users []model.User, count int64, err error) {
	return db.GetUsers(pageIndex, pageSize)
}

func CreateUser(u *model.User) error {
	// Role permission scopes use absolute paths; users always browse from /.
	u.BasePath = "/"
	return db.CreateUser(u)
}

func DeleteUserById(id uint) error {
	old, err := db.GetUserById(id)
	if err != nil {
		return err
	}
	if old.IsAdmin() || old.IsGuest() {
		return errs.DeleteAdminOrGuest
	}
	userCache.Del(old.Username)
	return db.DeleteUserById(id)
}

func UpdateUser(u *model.User) error {
	old, err := db.GetUserById(u.ID)
	if err != nil {
		return err
	}
	if u.IsGuest() {
		guestUser = nil
	}
	userCache.Del(old.Username)
	u.BasePath = "/"
	return db.UpdateUser(u)
}

// ClearUserCache drops users loaded before a startup data migration committed.
func ClearUserCache() {
	userCache.Clear()
	guestUser = nil
}

func Cancel2FAByUser(u *model.User) error {
	u.OtpSecret = ""
	return UpdateUser(u)
}

func Cancel2FAById(id uint) error {
	user, err := db.GetUserById(id)
	if err != nil {
		return err
	}
	return Cancel2FAByUser(user)
}

func DelUserCache(username string) error {
	user, err := GetUserByName(username)
	if err != nil {
		return err
	}
	if user.IsGuest() {
		guestUser = nil
	}
	userCache.Del(username)
	return nil
}

func CountEnabledAdminsExcluding(userID uint) (int64, error) {
	adminRole, err := GetRoleByName("admin")
	if err != nil {
		return 0, err
	}
	return db.CountUsersByRoleAndEnabledExclude(adminRole.ID, userID)
}
