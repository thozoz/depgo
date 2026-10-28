package sign

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	pkgSign "github.com/alist-org/alist/v3/pkg/sign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func initTestDb() {
	dB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic("failed to connect database")
	}
	conf.Conf = conf.DefaultConfig()
	db.Init(dB)
}

func TestSignEmitsFiniteLifetimeWithZeroOrUnsetSetting(t *testing.T) {
	initTestDb()
	_ = op.SaveSettingItem(&model.SettingItem{Key: conf.Token, Value: "test-site-token"})
	op.SettingCacheUpdate()
	Instance()

	for _, zeroVal := range []string{"0", "", "-1"} {
		t.Run("setting="+zeroVal, func(t *testing.T) {
			_ = op.SaveSettingItem(&model.SettingItem{Key: conf.LinkExpiration, Value: zeroVal})
			op.SettingCacheUpdate()

			s := Sign("/test/file.txt")
			assert.False(t, strings.HasSuffix(s, ":0"), "signature must not end with :0 perpetual marker")

			parts := strings.Split(s, ":")
			require.Len(t, parts, 2, "signature must have format <hash>:<expires>")
			exp, err := strconv.ParseInt(parts[1], 10, 64)
			require.NoError(t, err)

			now := time.Now().Unix()
			minExpected := now + int64((DefaultLinkExpirationHours-1)*3600)
			maxExpected := now + int64((DefaultLinkExpirationHours+1)*3600)
			assert.True(t, exp >= minExpected && exp <= maxExpected, "expected expiry ~%dh in future, got %d", DefaultLinkExpirationHours, exp-now)

			require.NoError(t, Verify("/test/file.txt", s))
		})
	}
}

func TestVerifyRejectsPerpetualSignatures(t *testing.T) {
	initTestDb()
	_ = op.SaveSettingItem(&model.SettingItem{Key: conf.Token, Value: "test-site-token"})
	op.SettingCacheUpdate()
	Instance()

	rawSigner := pkgSign.NewHMACSign([]byte("test-site-token"))
	perpetualSig := rawSigner.Sign("/secret/data.pdf", 0)
	require.True(t, strings.HasSuffix(perpetualSig, ":0"))

	err := Verify("/secret/data.pdf", perpetualSig)
	assert.ErrorIs(t, err, pkgSign.ErrSignExpired, "perpetual signatures must be rejected")
}

func TestVerifyRejectsExpiredSignatures(t *testing.T) {
	initTestDb()
	_ = op.SaveSettingItem(&model.SettingItem{Key: conf.Token, Value: "test-site-token"})
	op.SettingCacheUpdate()
	Instance()

	rawSigner := pkgSign.NewHMACSign([]byte("test-site-token"))
	expiredSig := rawSigner.Sign("/secret/data.pdf", time.Now().Add(-1*time.Hour).Unix())

	err := Verify("/secret/data.pdf", expiredSig)
	assert.ErrorIs(t, err, pkgSign.ErrSignExpired)
}
