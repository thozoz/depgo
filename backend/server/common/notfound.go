package common

import (
	"errors"
	"strings"

	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// IsNotFound reports whether err describes a resource that simply does not
// exist, as opposed to a failure while looking it up.
//
// A surprising share of the read endpoints used to answer 500 for a missing
// record, which tells a client "the server is broken" when the truthful answer
// is "no such object". That distinction matters to clients (retry vs give up)
// and to monitoring, where a 5xx is an alert.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}
	if errors.Is(err, errs.MetaNotFound) || errors.Is(err, errs.StorageNotFound) {
		return true
	}
	// The db layer wraps gorm errors with pkg/errors, and some call sites wrap
	// again with a custom message, so fall back to inspecting the text.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "record not found") ||
		strings.Contains(msg, "not found") && (strings.Contains(msg, "failed get") ||
			strings.Contains(msg, "failed find"))
}

// ErrorRespAuto responds with 404 when the error describes a missing resource
// and with the supplied fallback code otherwise.
func ErrorRespAuto(c *gin.Context, err error, fallback int, l ...bool) {
	if IsNotFound(err) {
		ErrorResp(c, err, 404, l...)
		return
	}
	ErrorResp(c, err, fallback, l...)
}

// GetErrResp is the shape used by the read-by-id handlers: a missing row is a
// 404, anything else is a 500.
func GetErrResp(c *gin.Context, err error, l ...bool) {
	ErrorRespAuto(c, err, 500, l...)
}
