package common

import (
	"fmt"
	"time"

	"github.com/Xhofe/go-cache"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

var SecretKey []byte

type UserClaims struct {
	Username string `json:"username"`
	PwdTS    int64  `json:"pwd_ts"`
	// DeviceKey is the device session this token was minted for. The middlewares
	// used to derive the session key from the client-supplied Client-Id header,
	// so a token for an evicted device could pick a different key by changing or
	// omitting that header, create a fresh active session, and keep working.
	DeviceKey string `json:"device_key"`
	jwt.RegisteredClaims
}

var validTokenCache = cache.NewMemCache[bool]()

// DefaultDeviceKey is the session key for flows that do not manage a client id
// (MCP automation, SSO, LDAP, WebAuthn). It matches the key the middlewares
// derive for a request without a Client-Id.
func DefaultDeviceKey(user *model.User) string {
	return utils.GetMD5EncodeStr(fmt.Sprintf("%d-", user.ID))
}

func GenerateToken(user *model.User) (tokenString string, err error) {
	return GenerateTokenWithDeviceKey(user, DefaultDeviceKey(user))
}

// GenerateTokenWithDeviceKey mints a token bound to a device session, so that
// deactivating that session cannot be escaped by changing the Client-Id header.
func GenerateTokenWithDeviceKey(user *model.User, deviceKey string) (tokenString string, err error) {
	claim := UserClaims{
		Username:  user.Username,
		PwdTS:     user.PwdTS,
		DeviceKey: deviceKey,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(conf.Conf.TokenExpiresIn) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	tokenString, err = token.SignedString(SecretKey)
	if err != nil {
		return "", err
	}
	validTokenCache.Set(tokenString, true)
	return tokenString, err
}

func ParseToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})
	if IsTokenInvalidated(tokenString) {
		return nil, errors.New("token is invalidated")
	}
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, errors.New("that's not even a token")
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, errors.New("token is expired")
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, errors.New("token not active yet")
			} else {
				return nil, errors.New("couldn't handle this token")
			}
		}
	}
	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("couldn't handle this token")
}

func InvalidateToken(tokenString string) error {
	if tokenString == "" {
		return nil // don't invalidate empty guest token
	}
	validTokenCache.Del(tokenString)
	return nil
}

func IsTokenInvalidated(tokenString string) bool {
	_, ok := validTokenCache.Get(tokenString)
	return !ok
}
