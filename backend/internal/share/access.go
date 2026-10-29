package share

import (
	"fmt"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/setting"
	signPkg "github.com/alist-org/alist/v3/pkg/sign"
)

// tokenPayload builds the signed string that binds an access token to one
// share.
//
// It deliberately contains no secret material. The token is handed to the
// holder of the password and travels in URLs, so anything inside it must be
// assumed readable by the client and by anyone who sees the link. Earlier
// versions embedded the share's PasswordHash, which leaked the (offline
// crackable) hash of the share password into every response, preview URL and
// browser history entry. The hash was not needed for verification either: the
// signature already authenticates the payload and AccessRevision already
// invalidates tokens whenever the password, expiry or enabled flag changes.
//
// ShareID keeps a token from being replayed against another share;
// AccessRevision keeps it from surviving a password change or a revocation.
//
// Consequence for upgrades: tokens minted before this change no longer verify,
// so holders of an old link are asked for the password once more. That is the
// intended trade — the alternative is keeping password hashes inside URLs.
func tokenPayload(share *model.Share) string {
	return fmt.Sprintf("%s:%d", share.ShareID, share.AccessRevision)
}

func signer() signPkg.Sign {
	return signPkg.NewHMACSign([]byte(setting.GetStr(conf.Token) + "-share-access"))
}

func SignAccess(share *model.Share, d time.Duration) string {
	return signer().Sign(tokenPayload(share), time.Now().Add(d).Unix())
}

func VerifyAccess(share *model.Share, token string) error {
	return signer().Verify(tokenPayload(share), token)
}
