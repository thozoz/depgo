package sign

import (
	"strings"
	"sync"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/pkg/sign"
)

const DefaultLinkExpirationHours = 4

var once sync.Once
var instance sign.Sign

func Sign(data string) string {
	expire := setting.GetInt(conf.LinkExpiration, DefaultLinkExpirationHours)
	if expire <= 0 {
		expire = DefaultLinkExpirationHours
	}
	return WithDuration(data, time.Duration(expire)*time.Hour)
}

func WithDuration(data string, d time.Duration) string {
	once.Do(Instance)
	return instance.Sign(data, time.Now().Add(d).Unix())
}

func NotExpired(data string) string {
	once.Do(Instance)
	return instance.Sign(data, 0)
}

func Verify(data string, s string) error {
	if strings.HasSuffix(s, ":0") {
		return sign.ErrSignExpired
	}
	once.Do(Instance)
	return instance.Verify(data, s)
}

func Instance() {
	instance = sign.NewHMACSign([]byte(setting.GetStr(conf.Token)))
}

