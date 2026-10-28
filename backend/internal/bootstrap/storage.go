package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/alist-org/alist/v3/cmd/flags"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/pkg/utils"
)

type defaultLocalAddition struct {
	RootFolderPath string `json:"root_folder_path"`
	StateFolder    string `json:"state_folder"`
	MkdirPerm      string `json:"mkdir_perm"`
}

func initDefaultLocalStorage() {
	rootPath := flags.RootFolder
	if rootPath == "" {
		rootPath = os.Getenv("DEPGO_LOCAL_ROOT")
	}
	if rootPath == "" {
		rootPath = filepath.Join(flags.DataDir, "storage")
	}
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		utils.Log.Fatalf("failed to resolve root path: %v", err)
	}
	if err := os.MkdirAll(absRoot, 0755); err != nil {
		utils.Log.Fatalf("failed to create root folder: %v", err)
	}

	stateFolder := filepath.Join(flags.DataDir, "localstate")
	absState, err := filepath.Abs(stateFolder)
	if err != nil {
		utils.Log.Fatalf("failed to resolve state path: %v", err)
	}
	if err := os.MkdirAll(absState, 0700); err != nil {
		utils.Log.Fatalf("failed to create state folder: %v", err)
	}

	addBytes, err := json.Marshal(defaultLocalAddition{
		RootFolderPath: absRoot,
		StateFolder:    absState,
		MkdirPerm:      "755",
	})
	if err != nil {
		utils.Log.Fatalf("failed to marshal local storage addition: %v", err)
	}

	storage := model.Storage{
		MountPath: "/",
		Driver:    "Local",
		Status:    "work",
		Addition:  string(addBytes),
		Disabled:  false,
	}
	if err := db.CreateStorage(&storage); err != nil {
		utils.Log.Errorf("failed to create default local storage: %v", err)
	} else {
		utils.Log.Infof("initialized default local storage at mount / pointing to %s", absRoot)
	}
}

func LoadStorages() {
	storages, err := db.GetEnabledStorages()
	if err != nil {
		utils.Log.Fatalf("failed get enabled storages: %+v", err)
	}
	if len(storages) == 0 {
		initDefaultLocalStorage()
		storages, _ = db.GetEnabledStorages()
	}

	go func(storages []model.Storage) {
		for i := range storages {
			err := op.LoadStorage(context.Background(), storages[i])
			if err != nil {
				utils.Log.Errorf("failed get enabled storages: %+v", err)
			} else {
				utils.Log.Infof("success load storage: [%s], driver: [%s], order: [%d]",
					storages[i].MountPath, storages[i].Driver, storages[i].Order)
			}
		}
		conf.StoragesLoaded = true
	}(storages)
}
