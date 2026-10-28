package sign

import (
	"strings"
	"sync"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/pkg/sign"
)

var onceArchive sync.Once
var instanceArchive sign.Sign

func SignArchive(data string) string {
	expire := setting.GetInt(conf.LinkExpiration, DefaultLinkExpirationHours)
	if expire <= 0 {
		expire = DefaultLinkExpirationHours
	}
	return WithDurationArchive(data, time.Duration(expire)*time.Hour)
}

func WithDurationArchive(data string, d time.Duration) string {
	onceArchive.Do(InstanceArchive)
	return instanceArchive.Sign(data, time.Now().Add(d).Unix())
}

func NotExpiredArchive(data string) string {
	onceArchive.Do(InstanceArchive)
	return instanceArchive.Sign(data, 0)
}

func VerifyArchive(data string, s string) error {
	if strings.HasSuffix(s, ":0") {
		return sign.ErrSignExpired
	}
	onceArchive.Do(InstanceArchive)
	return instanceArchive.Verify(data, s)
}

func InstanceArchive() {
	instanceArchive = sign.NewHMACSign([]byte(setting.GetStr(conf.Token) + "-archive"))
}

