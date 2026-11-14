package device

import (
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// CheckActive requires a live session; a deleted row cannot resurrect a bound token.
func CheckActive(userID uint, deviceKey string) error {
	sess, err := db.GetSession(userID, deviceKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errs.SessionInactive)
	}
	if err != nil {
		return err
	}
	ttl := setting.GetInt(conf.DeviceSessionTTL, 86400)
	if sess.Status != model.SessionActive || (ttl > 0 && sess.LastActive < time.Now().Unix()-int64(ttl)) {
		return errors.WithStack(errs.SessionInactive)
	}
	return nil
}

// HandleBound refreshes an issued session without enrolling a missing one.
func HandleBound(userID uint, deviceKey, ua, ip string) error {
	if err := CheckActive(userID, deviceKey); err != nil {
		return err
	}
	return db.RefreshActiveSession(userID, deviceKey, ua, utils.MaskIP(ip), time.Now().Unix())
}

// Handle verifies device sessions for a user and upserts current session.
func Handle(userID uint, deviceKey, ua, ip string) error {
	ttl := setting.GetInt(conf.DeviceSessionTTL, 86400)
	if ttl > 0 {
		_ = db.DeleteSessionsBefore(time.Now().Unix() - int64(ttl))
	}

	ip = utils.MaskIP(ip)

	now := time.Now().Unix()
	sess, err := db.GetSession(userID, deviceKey)
	if err == nil {
		if sess.Status == model.SessionInactive {
			return errors.WithStack(errs.SessionInactive)
		}
		// Refresh metadata without rewriting status: eviction may race this request.
		return db.RefreshActiveSession(userID, deviceKey, ua, ip, now)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	max := setting.GetInt(conf.MaxDevices, 0)
	evictOldest := setting.GetStr(conf.DeviceEvictPolicy, "deny") == "evict_oldest"

	s := &model.Session{UserID: userID, DeviceKey: deviceKey, UserAgent: ua, IP: ip, LastActive: now, Status: model.SessionActive}
	// Count and insert in one transaction: checking the budget separately let
	// concurrent logins all observe the same free slot and overshoot the limit.
	err = db.CreateSessionWithinLimit(s, int64(max), evictOldest)
	return err
}

// EnsureActiveOnLogin is used only in the login flow:
//   - If the session exists (even Inactive): reactivate it and refresh fields.
//   - If it does not exist: apply the max-devices policy, then create an Active
//     session.
//
// The budget check and the insert happen in one transaction. Doing them as two
// statements let concurrent logins from different clients all observe the same
// free slot, so a max_devices=2 account accumulated five active sessions.
func EnsureActiveOnLogin(userID uint, deviceKey, ua, ip string) error {
	ip = utils.MaskIP(ip)
	now := time.Now().Unix()

	maxDevices := int64(setting.GetInt(conf.MaxDevices, 0))
	evictOldest := setting.GetStr(conf.DeviceEvictPolicy, "deny") == "evict_oldest"

	sess, err := db.GetSession(userID, deviceKey)
	if err == nil {
		if sess.Status == model.SessionInactive {
			// Reactivating a retired session consumes a slot just like a new
			// one, so it goes through the same guarded path.
			reactivated := *sess
			reactivated.Status = model.SessionActive
			reactivated.LastActive = now
			reactivated.UserAgent = ua
			reactivated.IP = ip
			return db.CreateSessionWithinLimit(&reactivated, maxDevices, evictOldest)
		}
		// Refresh metadata without rewriting status: eviction may race this request.
		return db.RefreshActiveSession(userID, deviceKey, ua, ip, now)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return db.CreateSessionWithinLimit(&model.Session{
		UserID:     userID,
		DeviceKey:  deviceKey,
		UserAgent:  ua,
		IP:         ip,
		LastActive: now,
		Status:     model.SessionActive,
	}, maxDevices, evictOldest)
}

// Refresh updates last_active for the session.
func Refresh(userID uint, deviceKey string) {
	_ = db.UpdateSessionLastActive(userID, deviceKey, time.Now().Unix())
}
