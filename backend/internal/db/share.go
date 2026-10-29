package db

import (
	"errors"
	"time"

	"github.com/alist-org/alist/v3/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrShareConsumed reports that a share's access budget was already spent by the
// time this request tried to claim its slot. Eligibility is checked before the
// content is resolved, so without a check at claim time two concurrent requests
// for a burn-after-read link both observe "one access left" and both are served.
var ErrShareConsumed = errors.New("share has been consumed")

func GetShareByShareID(shareID string) (*model.Share, error) {
	var share model.Share
	if err := db.Where("share_id = ?", shareID).Take(&share).Error; err != nil {
		return nil, err
	}
	return &share, nil
}

func GetShareByCreatorAndShareID(creatorID uint, shareID string) (*model.Share, error) {
	var share model.Share
	if err := db.Where("creator_id = ? AND share_id = ?", creatorID, shareID).Take(&share).Error; err != nil {
		return nil, err
	}
	return &share, nil
}

func ShareIDExists(shareID string) (bool, error) {
	var count int64
	if err := db.Model(&model.Share{}).Where("share_id = ?", shareID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func ShareIDExistsExceptID(shareID string, id uint) (bool, error) {
	var count int64
	if err := db.Model(&model.Share{}).Where("share_id = ? AND id <> ?", shareID, id).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func CreateShare(share *model.Share) error {
	// Select all fields: GORM otherwise omits false booleans that carry a
	// `default:true` tag, silently re-enabling disabled share permissions.
	return db.Select("*").Create(share).Error
}

func UpdateShare(share *model.Share) error {
	return db.Save(share).Error
}

func GetSharesByCreator(creatorID uint, pageIndex, pageSize int) (shares []model.Share, count int64, err error) {
	tx := db.Model(&model.Share{}).Where("creator_id = ?", creatorID)
	err = tx.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = tx.Order("created_at desc").Offset((pageIndex - 1) * pageSize).Limit(pageSize).Find(&shares).Error
	return
}

func DeleteShareByShareID(creatorID uint, shareID string) error {
	return db.Where("creator_id = ? AND share_id = ?", creatorID, shareID).Delete(&model.Share{}).Error
}

func DisableShareByShareID(creatorID uint, shareID string) error {
	return db.Model(&model.Share{}).
		Where("creator_id = ? AND share_id = ?", creatorID, shareID).
		Updates(map[string]interface{}{
			"enabled":         false,
			"access_revision": gorm.Expr("access_revision + 1"),
		}).Error
}

// EnableShare turns a disabled share back on and returns it.
//
// It runs under the row lock so it cannot interleave with RecordShareAccess, and
// it always bumps access_revision: a token issued before the share was disabled
// stays dead instead of coming back to life with the link.
//
// A share that was switched off because its access budget ran out (burn after
// read, access_limit) would be disabled again by the very next request, so the
// owner enabling it again starts a fresh budget: the counter resets and the
// consumed marker is cleared. Enabling a share that is already on is a no-op.
func EnableShare(creatorID uint, shareID string) (*model.Share, error) {
	var share model.Share
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("creator_id = ? AND share_id = ?", creatorID, shareID).
			Take(&share).Error; err != nil {
			return err
		}
		if share.Enabled && !share.IsConsumed() {
			return nil
		}
		updates := map[string]interface{}{
			"enabled":         true,
			"access_revision": gorm.Expr("access_revision + 1"),
		}
		if share.IsConsumed() {
			updates["access_count"] = 0
			updates["consumed_at"] = nil
		}
		if err := tx.Model(&model.Share{}).Where("id = ?", share.ID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", share.ID).Take(&share).Error
	})
	if err != nil {
		return nil, err
	}
	return &share, nil
}

func TouchShareView(shareID string) error {
	now := time.Now()
	return db.Model(&model.Share{}).
		Where("share_id = ?", shareID).
		UpdateColumns(map[string]interface{}{
			"last_access_at": now,
			"view_count":     gorm.Expr("view_count + ?", 1),
		}).Error
}

func TouchShareDownload(shareID string) error {
	now := time.Now()
	return db.Model(&model.Share{}).
		Where("share_id = ?", shareID).
		UpdateColumns(map[string]interface{}{
			"last_access_at": now,
			"download_count": gorm.Expr("download_count + ?", 1),
		}).Error
}

func RecordShareAccess(shareID string) (*model.Share, error) {
	var updated model.Share
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("share_id = ?", shareID).
			Take(&updated).Error; err != nil {
			return err
		}

		// Re-check the budget under the row lock. The caller already checked
		// eligibility, but between that check and this write another request may
		// have spent the last access; claiming a slot that no longer exists must
		// fail rather than over-serve a burn-after-read link.
		if updated.IsConsumed() {
			return ErrShareConsumed
		}

		now := time.Now()
		updated.AccessCount++
		updated.LastAccessAt = &now
		updates := map[string]interface{}{
			"access_count":   updated.AccessCount,
			"last_access_at": now,
		}

		limit := updated.EffectiveAccessLimit()
		if limit > 0 && updated.AccessCount >= limit && updated.BurnAfterRead {
			// "Delete after reading": the owner opted into a self-destructing link,
			// so the share record goes now. The file itself is untouched. The copy
			// returned below still lets this one request be served.
			updated.Enabled = false
			updated.ConsumedAt = &now
			return tx.Delete(&model.Share{}, updated.ID).Error
		}
		if limit > 0 && updated.AccessCount >= limit {
			updated.Enabled = false
			updated.ConsumedAt = &now
			updates["enabled"] = false
			updates["consumed_at"] = now
		}

		return tx.Model(&model.Share{}).
			Where("id = ?", updated.ID).
			UpdateColumns(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}
