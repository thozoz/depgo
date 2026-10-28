package local

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	stdpath "path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/localstate"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/sign"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/alist-org/times"
	cp "github.com/otiai10/copy"
	_ "golang.org/x/image/webp"
)

type Local struct {
	model.Storage
	Addition
	mkdirPerm int32
	thumbSize int

	// zero means no limit
	thumbConcurrency int
	thumbTokenBucket TokenBucket

	// video thumb position
	videoThumbPos             float64
	videoThumbPosIsPercentage bool
	thumbPixel                int

	// use ffmpeg
	useFFmpeg bool

	state *localstate.Service
}

func (d *Local) Config() driver.Config {
	return config
}

func (d *Local) Init(ctx context.Context) error {
	if d.MkdirPerm == "" {
		d.mkdirPerm = 0777
	} else {
		v, err := strconv.ParseUint(d.MkdirPerm, 8, 32)
		if err != nil {
			return err
		}
		d.mkdirPerm = int32(v)
	}
	if !utils.Exists(d.GetRootPath()) {
		return fmt.Errorf("root folder %s not exists", d.GetRootPath())
	}
	if !filepath.IsAbs(d.GetRootPath()) {
		abs, err := filepath.Abs(d.GetRootPath())
		if err != nil {
			return err
		}
		d.Addition.RootFolderPath = abs
	}

	d.useFFmpeg = d.UseFFmpeg

	if d.ThumbCacheFolder != "" && !utils.Exists(d.ThumbCacheFolder) {
		err := os.MkdirAll(d.ThumbCacheFolder, os.FileMode(d.mkdirPerm))
		if err != nil {
			return err
		}
	}
	d.thumbSize = 144
	if item, err := op.GetSettingItemByKey(conf.ThumbnailSize); err == nil && item != nil && strings.TrimSpace(item.Value) != "" {
		v, err := strconv.ParseUint(item.Value, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid setting %s value: %s, err: %s", conf.ThumbnailSize, item.Value, err)
		}
		if v == 0 {
			return fmt.Errorf("invalid setting %s value: %s, the value must be a positive integer", conf.ThumbnailSize, item.Value)
		}
		d.thumbSize = int(v)
	}
	if d.ThumbConcurrency != "" {
		v, err := strconv.ParseUint(d.ThumbConcurrency, 10, 32)
		if err != nil {
			return err
		}
		d.thumbConcurrency = int(v)
	}
	if d.ThumbPixel != "" {
		v, err := strconv.ParseUint(d.ThumbPixel, 10, 32)
		if err != nil {
			return err
		}
		d.thumbPixel = int(v)
	}

	if d.thumbConcurrency == 0 {
		d.thumbTokenBucket = NewNopTokenBucket()
	} else {
		d.thumbTokenBucket = NewStaticTokenBucketWithMigration(d.thumbTokenBucket, d.thumbConcurrency)
	}
	// Check the VideoThumbPos value
	if d.VideoThumbPos == "" {
		d.VideoThumbPos = "20%"
	}
	if strings.HasSuffix(d.VideoThumbPos, "%") {
		percentage := strings.TrimSuffix(d.VideoThumbPos, "%")
		val, err := strconv.ParseFloat(percentage, 64)
		if err != nil {
			return fmt.Errorf("invalid video_thumb_pos value: %s, err: %s", d.VideoThumbPos, err)
		}
		if val < 0 || val > 100 {
			return fmt.Errorf("invalid video_thumb_pos value: %s, the precentage must be a number between 0 and 100", d.VideoThumbPos)
		}
		d.videoThumbPosIsPercentage = true
		d.videoThumbPos = val / 100
	} else {
		val, err := strconv.ParseFloat(d.VideoThumbPos, 64)
		if err != nil {
			return fmt.Errorf("invalid video_thumb_pos value: %s, err: %s", d.VideoThumbPos, err)
		}
		if val < 0 {
			return fmt.Errorf("invalid video_thumb_pos value: %s, the time must be a positive number", d.VideoThumbPos)
		}
		d.videoThumbPosIsPercentage = false
		d.videoThumbPos = val
	}

	if d.state != nil {
		_ = d.state.Close()
		d.state = nil
	}

	svc, err := localstate.New(localstate.Config{
		StorageID: d.Storage.ID,
		RootPath:  d.GetRootPath(),
		StatePath: d.Addition.StateFolder,
		DB:        db.GetDb(),
	})
	if err != nil {
		return fmt.Errorf("init localstate service: %w", err)
	}
	d.state = svc

	return nil
}

func (d *Local) Drop(ctx context.Context) error {
	if d.state != nil {
		_ = d.state.Close()
		d.state = nil
	}
	return nil
}

func (d *Local) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *Local) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	fullPath := dir.GetPath()
	if d.state != nil {
		rel, err := d.state.DiskToRelative(fullPath)
		if err != nil {
			return nil, err
		}
		if err := d.state.CheckSymlink(rel); err != nil {
			return nil, err
		}
	}
	rawFiles, err := readDir(fullPath)
	if err != nil {
		return nil, err
	}
	var files []model.Obj
	for _, f := range rawFiles {
		if !d.ShowHidden && strings.HasPrefix(f.Name(), ".") {
			continue
		}
		// Skip symlinks to prevent symlink traversal/leakage in listings
		if f.Mode()&os.ModeSymlink != 0 {
			continue
		}
		file := d.FileInfoToObj(ctx, f, args.ReqPath, fullPath)
		files = append(files, file)
	}
	return files, nil
}
func (d *Local) FileInfoToObj(ctx context.Context, f fs.FileInfo, reqPath string, fullPath string) model.Obj {
	thumb := ""
	if d.Thumbnail {
		typeName := utils.GetFileType(f.Name())
		if typeName == conf.IMAGE || typeName == conf.VIDEO {
			thumb = common.GetApiUrl(common.GetHttpReq(ctx)) + stdpath.Join("/d", reqPath, f.Name())
			thumb = utils.EncodePath(thumb, true)
			thumb += "?type=thumb&sign=" + sign.Sign(stdpath.Join(reqPath, f.Name()))
		}
	}
	filePath := filepath.Join(fullPath, f.Name())
	isFolder := f.IsDir() || isLinkedDir(f, filePath)
	var size int64
	if !isFolder {
		size = f.Size()
	}
	var ctime time.Time
	t, err := times.Stat(filePath)
	if err == nil {
		if t.HasBirthTime() {
			ctime = t.BirthTime()
		}
	}

	file := model.ObjThumb{
		Object: model.Object{
			Path:     filePath,
			Name:     f.Name(),
			Modified: f.ModTime(),
			Size:     size,
			IsFolder: isFolder,
			Ctime:    ctime,
		},
		Thumbnail: model.Thumbnail{
			Thumbnail: thumb,
		},
	}
	return &file
}
func (d *Local) GetMeta(ctx context.Context, path string) (model.Obj, error) {
	f, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	file := d.FileInfoToObj(ctx, f, path, path)
	//h := "123123"
	//if s, ok := f.(model.SetHash); ok && file.GetHash() == ("","")  {
	//	s.SetHash(h,"SHA1")
	//}
	return file, nil

}

func (d *Local) Get(ctx context.Context, path string) (model.Obj, error) {
	var f fs.FileInfo
	var err error
	var fullPath string

	if d.state != nil {
		rel, errRel := d.state.CanonicalRelPath(path)
		if errRel != nil {
			return nil, errs.ObjectNotFound
		}
		if errSym := d.state.CheckSymlink(rel); errSym != nil {
			return nil, errSym
		}
		target := rel
		if target == "" {
			target = "."
		}
		f, err = d.state.OsRoot().Stat(target)
		fullPath, _ = d.state.RelToDiskPath(rel)
	} else {
		fullPath = filepath.Join(d.GetRootPath(), path)
		f, err = os.Stat(fullPath)
	}

	if err != nil {
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "cannot find the file") {
			return nil, errs.ObjectNotFound
		}
		return nil, err
	}
	isFolder := f.IsDir() || isLinkedDir(f, fullPath)
	size := f.Size()
	if isFolder {
		size = 0
	}
	var ctime time.Time
	t, err := times.Stat(fullPath)
	if err == nil {
		if t.HasBirthTime() {
			ctime = t.BirthTime()
		}
	}
	file := model.Object{
		Path:     fullPath,
		Name:     f.Name(),
		Modified: f.ModTime(),
		Ctime:    ctime,
		Size:     size,
		IsFolder: isFolder,
	}
	return &file, nil
}

func (d *Local) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	fullPath := file.GetPath()
	var link model.Link
	if args.Type == "thumb" && utils.Ext(file.GetName()) != "svg" {
		var buf *bytes.Buffer
		var thumbPath *string
		err := d.thumbTokenBucket.Do(ctx, func() error {
			var err error
			buf, thumbPath, err = d.getThumb(file)
			return err
		})
		if err != nil {
			return nil, err
		}
		link.Header = http.Header{
			"Content-Type": []string{"image/png"},
		}
		if thumbPath != nil {
			open, err := os.Open(*thumbPath)
			if err != nil {
				return nil, err
			}
			link.MFile = open
		} else {
			link.MFile = model.NewNopMFile(bytes.NewReader(buf.Bytes()))
		}
	} else {
		var open *os.File
		var err error
		if d.state != nil {
			rel, errRel := d.state.DiskToRelative(fullPath)
			if errRel != nil {
				return nil, errRel
			}
			if errSym := d.state.CheckSymlink(rel); errSym != nil {
				return nil, errSym
			}
			open, err = d.state.OsRoot().Open(rel)
		} else {
			open, err = os.Open(fullPath)
		}
		if err != nil {
			return nil, err
		}
		link.MFile = open
	}
	return &link, nil
}

func (d *Local) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) error {
	ctx = localstate.NormalizeContext(ctx)
	if d.state != nil {
		return d.state.SafeMakeDir(ctx, parentDir, dirName, os.FileMode(d.mkdirPerm))
	}
	fullPath := filepath.Join(parentDir.GetPath(), dirName)
	err := os.MkdirAll(fullPath, os.FileMode(d.mkdirPerm))
	if err != nil {
		return err
	}
	return nil
}

func (d *Local) Move(ctx context.Context, srcObj, dstDir model.Obj) error {
	ctx = localstate.NormalizeContext(ctx)
	if d.state != nil {
		return d.state.SafeMove(ctx, srcObj, dstDir, d.Storage.MountPath)
	}
	srcPath := srcObj.GetPath()
	dstPath := filepath.Join(dstDir.GetPath(), srcObj.GetName())
	if utils.IsSubPath(srcPath, dstPath) {
		return fmt.Errorf("the destination folder is a subfolder of the source folder")
	}
	return os.Rename(srcPath, dstPath)
}

func (d *Local) Rename(ctx context.Context, srcObj model.Obj, newName string) error {
	ctx = localstate.NormalizeContext(ctx)
	if d.state != nil {
		return d.state.SafeRename(ctx, srcObj, newName, d.Storage.MountPath)
	}
	srcPath := srcObj.GetPath()
	dstPath := filepath.Join(filepath.Dir(srcPath), newName)
	return os.Rename(srcPath, dstPath)
}

func (d *Local) Copy(ctx context.Context, srcObj, dstDir model.Obj) error {
	ctx = localstate.NormalizeContext(ctx)
	if d.state != nil {
		return d.state.SafeCopy(ctx, srcObj, dstDir)
	}
	srcPath := srcObj.GetPath()
	dstPath := filepath.Join(dstDir.GetPath(), srcObj.GetName())
	if utils.IsSubPath(srcPath, dstPath) {
		return fmt.Errorf("the destination folder is a subfolder of the source folder")
	}
	return cp.Copy(srcPath, dstPath, cp.Options{
		Sync:          true,
		PreserveTimes: true,
		PreserveOwner: true,
	})
}

func (d *Local) Remove(ctx context.Context, obj model.Obj) error {
	ctx = localstate.NormalizeContext(ctx)
	if d.state != nil {
		rel, err := d.state.DiskToRelative(obj.GetPath())
		if err != nil {
			return err
		}
		_, err = d.state.MoveToTrash(ctx, rel)
		return err
	}
	if obj.IsDir() {
		return os.RemoveAll(obj.GetPath())
	}
	return os.Remove(obj.GetPath())
}

func (d *Local) Put(ctx context.Context, dstDir model.Obj, stream model.FileStreamer, up driver.UpdateProgress) error {
	ctx = localstate.NormalizeContext(ctx)
	if d.state != nil {
		return d.state.SafePut(ctx, dstDir, stream, up)
	}
	fullPath := filepath.Join(dstDir.GetPath(), stream.GetName())
	out, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
		if errors.Is(err, context.Canceled) {
			_ = os.Remove(fullPath)
		}
	}()
	err = utils.CopyWithCtx(ctx, out, stream, stream.GetSize(), up)
	if err != nil {
		return err
	}
	_ = os.Chtimes(fullPath, stream.ModTime(), stream.ModTime())
	return nil
}

func (d *Local) State() *localstate.Service {
	return d.state
}

func (d *Local) SetState(s *localstate.Service) {
	d.state = s
}

func (d *Local) ListTrash(ctx context.Context) ([]model.TrashItem, error) {
	if d.state == nil {
		return nil, errors.New("local state not initialized")
	}
	return d.state.ListTrash(localstate.NormalizeContext(ctx))
}

func (d *Local) RestoreTrash(ctx context.Context, trashID string, customTarget string) (*model.FileRecord, error) {
	if d.state == nil {
		return nil, errors.New("local state not initialized")
	}
	return d.state.RestoreFromTrash(localstate.NormalizeContext(ctx), trashID, customTarget)
}

func (d *Local) PurgeTrash(ctx context.Context, trashID string) error {
	if d.state == nil {
		return errors.New("local state not initialized")
	}
	return d.state.PurgeTrash(localstate.NormalizeContext(ctx), trashID)
}

func (d *Local) ListVersions(ctx context.Context, relPath string) ([]model.FileVersion, error) {
	if d.state == nil {
		return nil, errors.New("local state not initialized")
	}
	ctx = localstate.NormalizeContext(ctx)
	rec, err := d.state.GetActiveFileRecord(ctx, relPath)
	if err != nil {
		return nil, err
	}
	return d.state.ListFileVersions(ctx, rec.FileID)
}

func (d *Local) RestoreVersion(ctx context.Context, fileID, versionID string, expectedRevision int64) (*model.FileRecord, error) {
	if d.state == nil {
		return nil, errors.New("local state not initialized")
	}
	return d.state.RestoreVersion(localstate.NormalizeContext(ctx), fileID, versionID, expectedRevision)
}

func (d *Local) GetRecord(ctx context.Context, relPath string) (*model.FileRecord, error) {
	if d.state == nil {
		return nil, errors.New("local state not initialized")
	}
	return d.state.GetCurrentRecord(localstate.NormalizeContext(ctx), relPath)
}

func (d *Local) ReadText(ctx context.Context, relPath string) (string, int64, error) {
	if d.state == nil {
		return "", 0, errors.New("local state not initialized")
	}
	return d.state.ReadText(localstate.NormalizeContext(ctx), relPath)
}

var _ driver.Driver = (*Local)(nil)
