package db

import (
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetSession(userID uint, deviceKey string) (*model.Session, error) {
	s := model.Session{UserID: userID, DeviceKey: deviceKey}
	if err := db.Select("user_id, device_key, last_active, status, user_agent, ip").Where(&s).First(&s).Error; err != nil {
		return nil, errors.Wrap(err, "failed find session")
	}
	return &s, nil
}

func CreateSession(s *model.Session) error {
	return errors.WithStack(db.Create(s).Error)
}

func UpsertSession(s *model.Session) error {
	return errors.WithStack(db.Clauses(clause.OnConflict{UpdateAll: true}).Create(s).Error)
}

func DeleteSession(userID uint, deviceKey string) error {
	return errors.WithStack(db.Where("user_id = ? AND device_key = ?", userID, deviceKey).Delete(&model.Session{}).Error)
}

// enrolSessionWithinLimit makes s an active session without ever letting the
// user exceed maxDevices, checking the budget and writing inside one
// transaction.
//
// device.Handle and EnsureActiveOnLogin used to count the active sessions and
// then create or reactivate one as separate statements. Concurrent logins from
// different clients all observed the same "one slot free" and all proceeded, so
// a max_devices=2 account accumulated five active sessions. The limit is a
// policy control against account sharing, so it has to hold under concurrency.
//
// alreadyActive says whether s is already counted in the active total (i.e. it
// exists and is currently active), in which case the budget does not change.
// evictOldest reproduces the "evict_oldest" policy: when the budget is exhausted
// the least recently active session is retired instead of refusing the login.
// The returned device key names the session that was evicted, if any.
func enrolSessionWithinLimit(s *model.Session, alreadyActive bool, maxDevices int64, evictOldest bool) (evictedDeviceKey string, err error) {
	err = db.Transaction(func(tx *gorm.DB) error {
		if maxDevices > 0 && !alreadyActive {
			var count int64
			if err := tx.Model(&model.Session{}).
				Where("user_id = ? AND status = ?", s.UserID, model.SessionActive).
				Count(&count).Error; err != nil {
				return errors.WithStack(err)
			}
			if count >= maxDevices {
				if !evictOldest {
					return errors.WithStack(errs.TooManyDevices)
				}
				var oldest model.Session
				if err := tx.Where("user_id = ? AND status = ?", s.UserID, model.SessionActive).
					Order("last_active ASC").Take(&oldest).Error; err != nil {
					return errors.WithStack(err)
				}
				if err := tx.Model(&model.Session{}).
					Where("device_key = ?", oldest.DeviceKey).
					Update("status", model.SessionInactive).Error; err != nil {
					return errors.WithStack(err)
				}
				evictedDeviceKey = oldest.DeviceKey
			}
		}
		return tx.Save(s).Error
	})
	if err != nil {
		return "", err
	}
	return evictedDeviceKey, nil
}

// CreateSessionWithinLimit writes s as an active session subject to the device
// limit. It covers both a brand-new device and a retired session being
// reactivated: either way a slot is consumed, and Save inserts or updates
// accordingly.
func CreateSessionWithinLimit(s *model.Session, maxDevices int64, evictOldest bool) error {
	_, err := enrolSessionWithinLimit(s, false, maxDevices, evictOldest)
	return err
}

func CountActiveSessionsByUser(userID uint) (int64, error) {
	var count int64
	err := db.Model(&model.Session{}).
		Where("user_id = ? AND status = ?", userID, model.SessionActive).
		Count(&count).Error
	return count, errors.WithStack(err)
}

func DeleteSessionsBefore(ts int64) error {
	return errors.WithStack(db.Where("last_active < ?", ts).Delete(&model.Session{}).Error)
}

// GetOldestActiveSession returns the oldest active session for the specified user.
func GetOldestActiveSession(userID uint) (*model.Session, error) {
	var s model.Session
	if err := db.Where("user_id = ? AND status = ?", userID, model.SessionActive).
		Order("last_active ASC").First(&s).Error; err != nil {
		return nil, errors.Wrap(err, "failed get oldest active session")
	}
	return &s, nil
}

func UpdateSessionLastActive(userID uint, deviceKey string, lastActive int64) error {
	return errors.WithStack(db.Model(&model.Session{}).Where("user_id = ? AND device_key = ?", userID, deviceKey).Update("last_active", lastActive).Error)
}

func ListSessionsByUser(userID uint) ([]model.Session, error) {
	var sessions []model.Session
	err := db.Select("user_id, device_key, last_active, status, user_agent, ip").Where("user_id = ? AND status = ?", userID, model.SessionActive).Find(&sessions).Error
	return sessions, errors.WithStack(err)
}

func ListSessions() ([]model.Session, error) {
	var sessions []model.Session
	err := db.Select("user_id, device_key, last_active, status, user_agent, ip").Where("status = ?", model.SessionActive).Find(&sessions).Error
	return sessions, errors.WithStack(err)
}

func MarkInactive(sessionID string) error {
	return errors.WithStack(db.Model(&model.Session{}).Where("device_key = ?", sessionID).Update("status", model.SessionInactive).Error)
}
