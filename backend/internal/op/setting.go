package op

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Xhofe/go-cache"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/singleflight"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/pkg/errors"
)

var settingCache = cache.NewMemCache(cache.WithShards[*model.SettingItem](4))
var settingG singleflight.Group[*model.SettingItem]
var settingCacheF = func(item *model.SettingItem) {
	settingCache.Set(item.Key, item, cache.WithEx[*model.SettingItem](time.Hour))
}

var settingGroupCache = cache.NewMemCache(cache.WithShards[[]model.SettingItem](4))
var settingGroupG singleflight.Group[[]model.SettingItem]
var settingGroupCacheF = func(key string, item []model.SettingItem) {
	settingGroupCache.Set(key, item, cache.WithEx[[]model.SettingItem](time.Hour))
}

var settingChangingCallbacks = make([]func(), 0)

func RegisterSettingChangingCallback(f func()) {
	settingChangingCallbacks = append(settingChangingCallbacks, f)
}

func SettingCacheUpdate() {
	settingCache.Clear()
	settingGroupCache.Clear()
	for _, cb := range settingChangingCallbacks {
		cb()
	}
}

// publicSettingAllowList names the settings a client may read without
// authenticating. It is an allow-list on purpose: a new internal setting must
// be a deliberate decision to expose, never an accidental inheritance of the
// "public" flag.
//
// Excluded on purpose, despite living next to the entries below:
//   - version: turns a generic deployment into a CVE target list entry;
//   - hide_files, privacy_regs, filename_char_mapping: describe the defenses
//     around content the operator chose to hide;
//   - ocr_api, forward/ignore_direct_link_params: internal plumbing;
//   - max_devices, device_*: account-policy internals;
//   - customize_head/customize_body: raw operator markup, not a client input.
var publicSettingAllowList = map[string]struct{}{
	// Branding and basic site metadata, needed before a user logs in.
	conf.SiteTitle:                {},
	conf.Logo:                     {},
	conf.Favicon:                  {},
	conf.MainColor:                {},
	conf.Announcement:             {},
	conf.AllowIndexed:             {},
	conf.AllowMounted:             {},
	conf.AllowRegister:            {},
	conf.DefaultRole:              {},
	conf.UseNewui:                 {},
	conf.FrontendRememberSort:     {},
	conf.AudioAutoplay:            {},
	conf.VideoAutoplay:            {},
	conf.PreviewArchivesByDefault: {},
	conf.ReadMeAutoRender:         {},
	conf.FilterReadMeScripts:      {},
	conf.ThumbnailSize:            {},
	// Login-page feature switches: the SPA must know which buttons to render.
	conf.WebauthnLoginEnabled: {},
	conf.SSOLoginEnabled:      {},
	conf.SSOLoginPlatform:     {},
	conf.SSOCompatibilityMode: {},
	conf.LdapLoginEnabled:     {},
	conf.LdapLoginTips:        {},
}

// GetPublicSettingsMap returns the subset of settings that unauthenticated
// clients are allowed to see.
func GetPublicSettingsMap() map[string]string {
	items, _ := GetPublicSettingItems()
	pSettings := make(map[string]string, len(items))
	for _, item := range items {
		if _, ok := publicSettingAllowList[item.Key]; !ok {
			continue
		}
		pSettings[item.Key] = item.Value
	}
	return pSettings
}

func GetSettingsMap() map[string]string {
	items, _ := GetSettingItems()
	settings := make(map[string]string)
	for _, item := range items {
		settings[item.Key] = item.Value
	}
	return settings
}

func GetSettingItems() ([]model.SettingItem, error) {
	if items, ok := settingGroupCache.Get("ALL_SETTING_ITEMS"); ok {
		return items, nil
	}
	items, err, _ := settingGroupG.Do("ALL_SETTING_ITEMS", func() ([]model.SettingItem, error) {
		_items, err := db.GetSettingItems()
		if err != nil {
			return nil, err
		}
		settingGroupCacheF("ALL_SETTING_ITEMS", _items)
		return _items, nil
	})
	return items, err
}

func GetPublicSettingItems() ([]model.SettingItem, error) {
	if items, ok := settingGroupCache.Get("ALL_PUBLIC_SETTING_ITEMS"); ok {
		return items, nil
	}
	items, err, _ := settingGroupG.Do("ALL_PUBLIC_SETTING_ITEMS", func() ([]model.SettingItem, error) {
		_items, err := db.GetPublicSettingItems()
		if err != nil {
			return nil, err
		}
		settingGroupCacheF("ALL_PUBLIC_SETTING_ITEMS", _items)
		return _items, nil
	})
	return items, err
}

func GetSettingItemByKey(key string) (*model.SettingItem, error) {
	if item, ok := settingCache.Get(key); ok {
		return item, nil
	}

	item, err, _ := settingG.Do(key, func() (*model.SettingItem, error) {
		_item, err := db.GetSettingItemByKey(key)
		if err != nil {
			return nil, err
		}
		settingCacheF(_item)
		return _item, nil
	})
	return item, err
}

func GetSettingItemInKeys(keys []string) ([]model.SettingItem, error) {
	var items []model.SettingItem
	for _, key := range keys {
		item, err := GetSettingItemByKey(key)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, nil
}

func GetSettingItemsByGroup(group int) ([]model.SettingItem, error) {
	key := strconv.Itoa(group)
	if items, ok := settingGroupCache.Get(key); ok {
		return items, nil
	}
	items, err, _ := settingGroupG.Do(key, func() ([]model.SettingItem, error) {
		_items, err := db.GetSettingItemsByGroup(group)
		if err != nil {
			return nil, err
		}
		settingGroupCacheF(key, _items)
		return _items, nil
	})
	return items, err
}

func GetSettingItemsInGroups(groups []int) ([]model.SettingItem, error) {
	sort.Ints(groups)
	key := strings.Join(utils.MustSliceConvert(groups, func(i int) string {
		return strconv.Itoa(i)
	}), ",")

	if items, ok := settingGroupCache.Get(key); ok {
		return items, nil
	}
	items, err, _ := settingGroupG.Do(key, func() ([]model.SettingItem, error) {
		_items, err := db.GetSettingItemsInGroups(groups)
		if err != nil {
			return nil, err
		}
		settingGroupCacheF(key, _items)
		return _items, nil
	})
	return items, err
}

func SaveSettingItems(items []model.SettingItem) error {
	noHookItems := make([]model.SettingItem, 0)
	errs := make([]error, 0)
	for i := range items {
		if ok, err := HandleSettingItemHook(&items[i]); ok {
			if err != nil {
				errs = append(errs, err)
			} else {
				err = db.SaveSettingItem(&items[i])
				if err != nil {
					errs = append(errs, err)
				}
			}
		} else {
			noHookItems = append(noHookItems, items[i])
		}
	}
	if len(noHookItems) > 0 {
		err := db.SaveSettingItems(noHookItems)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) < len(items)-len(noHookItems)+1 {
		SettingCacheUpdate()
	}
	return utils.MergeErrors(errs...)
}

// SetSettingValue updates only the value of an existing setting, keeping the
// rest of the row (type, group, flag, help, options, display index, recorded
// default) exactly as it was.
//
// Building a fresh SettingItem with just the new value looks equivalent but is
// not: db.Save writes every column except the ones gorm is told to skip, so the
// omitted columns were reset to their zero values. For the site token that meant
// losing the display index (the row jumped to the top of the admin settings
// list), the value type, the help text and the visibility flag - all of which
// the admin settings page reads back.
func SetSettingValue(key, value string) error {
	item, err := GetSettingItemByKey(key)
	if err != nil {
		return errors.WithMessagef(err, "failed to get setting [%s]", key)
	}
	item.Value = value
	return SaveSettingItem(item)
}

func SaveSettingItem(item *model.SettingItem) (err error) {
	// hook
	if _, err := HandleSettingItemHook(item); err != nil {
		return err
	}
	// update
	if err = db.SaveSettingItem(item); err != nil {
		return err
	}
	SettingCacheUpdate()
	return nil
}

// SetSettingValues updates the values of several existing settings at once,
// leaving their other columns alone. See SetSettingValue for why the metadata
// must be preserved.
func SetSettingValues(values map[string]string) error {
	merged := make([]model.SettingItem, 0, len(values))
	for key, value := range values {
		item, err := GetSettingItemByKey(key)
		if err != nil {
			return errors.WithMessagef(err, "failed to get setting [%s]", key)
		}
		item.Value = value
		merged = append(merged, *item)
	}
	return SaveSettingItems(merged)
}

func DeleteSettingItemByKey(key string) error {
	old, err := GetSettingItemByKey(key)
	if err != nil {
		return errors.WithMessage(err, "failed to get settingItem")
	}
	if !old.IsDeprecated() {
		return errors.Errorf("setting [%s] is not deprecated", key)
	}
	SettingCacheUpdate()
	return db.DeleteSettingItemByKey(key)
}
