package handles

import (
	"time"

	"github.com/alist-org/alist/v3/internal/diskusage"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
)

var usageSizer = &diskusage.Sizer{TTL: 20 * time.Second}

type UsageResp struct {
	// Total and Free describe the filesystem the served root lives on.
	Total int64 `json:"total"`
	Free  int64 `json:"free"`
	// Files is what the served tree holds; State is what trash and file
	// versions hold. Used is their sum, the space Depgo is responsible for.
	Files int64 `json:"files"`
	State int64 `json:"state"`
	Used  int64 `json:"used"`
}

// FsUsage reports disk capacity and how much of it Depgo itself occupies.
func FsUsage(c *gin.Context) {
	storage, _, err := getLocalStorage(c, "/")
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	state := storage.State()
	root := state.RootPath()
	total, free, err := diskusage.Filesystem(root)
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	files := usageSizer.Size(root)
	stateBytes := usageSizer.Size(state.StatePath())
	common.SuccessResp(c, UsageResp{
		Total: int64(total),
		Free:  int64(free),
		Files: files,
		State: stateBytes,
		Used:  files + stateBytes,
	})
}
