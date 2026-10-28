package data

import (
	"os"
	"path/filepath"

	"github.com/alist-org/alist/v3/cmd/flags"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/pkg/utils/random"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func initUser() {
	guest, err := op.GetGuest()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			salt := random.String(16)
			guestRole, _ := op.GetRoleByName("guest")
			guest = &model.User{
				Username:   "guest",
				PwdHash:    model.TwoHashPwd("guest", salt),
				Salt:       salt,
				Role:       model.Roles{int(guestRole.ID)},
				BasePath:   "/",
				Permission: 0,
				Disabled:   true,
				Authn:      "[]",
			}
			if err := db.CreateUser(guest); err != nil {
				utils.Log.Fatalf("[init user] Failed to create guest user: %v", err)
			}
		} else {
			utils.Log.Fatalf("[init user] Failed to get guest user: %v", err)
		}
	}
	admin, err := op.GetAdmin()
	adminPassword := random.String(12)
	envpass := os.Getenv("DEPGO_ADMIN_PASSWORD")
	if envpass == "" {
		envpass = os.Getenv("ALIST_ADMIN_PASSWORD")
	}
	if flags.Dev {
		adminPassword = "admin"
	} else if len(envpass) > 0 {
		adminPassword = envpass
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			salt := random.String(16)
			adminRole, _ := op.GetRoleByName("admin")
			admin = &model.User{
				Username: "admin",
				Salt:     salt,
				PwdHash:  model.TwoHashPwd(adminPassword, salt),
				Role:     model.Roles{int(adminRole.ID)},
				BasePath: "/",
				Authn:    "[]",
				// 0(can see hidden) - 7(can remove) & 12(can read archives) - 13(can decompress archives)
				Permission: 0xFFFF,
			}
			if err := op.CreateUser(admin); err != nil {
				panic(err)
			} else {
				if envpass != "" {
					utils.Log.Infof("Successfully created admin user from environment credentials")
				} else {
					credPath := filepath.Join(flags.DataDir, ".initial_admin_password")
					_ = os.WriteFile(credPath, []byte(adminPassword+"\n"), 0600)
					utils.Log.Infof("Successfully created admin user (initial credentials stored securely in %s)", credPath)
				}
			}
		} else {
			utils.Log.Fatalf("[init user] Failed to get admin user: %v", err)
		}
	}
}
