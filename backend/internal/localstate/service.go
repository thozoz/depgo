package localstate

import (
	"context"
	"errors"
	"fmt"
	"os"
	stdpath "path"
	"path/filepath"
	"strings"
	"sync"

	"gorm.io/gorm"
)

var (
	ErrDestinationExists = errors.New("destination already exists")
	ErrNotFound          = errors.New("file record not found")
	ErrSymlinkForbidden  = errors.New("symlink traversal not permitted in local storage")
	ErrPathEscapesRoot   = errors.New("path escapes storage root")
	ErrNestedStateRoot   = errors.New("local state folder cannot be nested inside or parent of storage root")
	ErrInvalidName       = errors.New("invalid path or file name component")
)

// Service coordinates lifecycle, versioning, trash, and crash-resilient operations for a local storage.
type Service struct {
	storageID   uint
	rootPath    string // canonical absolute path to served root
	statePath   string // canonical absolute path to state root
	stagingPath string // canonical absolute path to state/staging
	trashPath   string // canonical absolute path to state/trash
	versionPath string // canonical absolute path to state/versions

	osRoot    *os.Root // anchored to rootPath
	stateRoot *os.Root // anchored to statePath

	db *gorm.DB

	// shortcut: storage-level mutex for all mutations, upgrade to per-file hierarchy locking if multi-user concurrency bottlenecked
	mu sync.Mutex
}

// Config specifies initialization parameters for the Service.
type Config struct {
	StorageID uint
	RootPath  string
	StatePath string
	DB        *gorm.DB
}

// New creates and initializes a new localstate Service.
func New(cfg Config) (*Service, error) {
	if cfg.RootPath == "" {
		return nil, fmt.Errorf("root path is required")
	}

	cleanRoot, err := canonicalizePath(cfg.RootPath)
	if err != nil {
		return nil, fmt.Errorf("canonicalize root path [%s]: %w", cfg.RootPath, err)
	}

	statePath := cfg.StatePath
	if statePath == "" {
		parent := filepath.Dir(cleanRoot)
		baseName := filepath.Base(cleanRoot)
		statePath = filepath.Join(parent, fmt.Sprintf(".depgo_state_%s_%d", baseName, cfg.StorageID))
	} else {
		statePath, err = filepath.Abs(statePath)
		if err != nil {
			return nil, fmt.Errorf("resolve abs state [%s]: %w", statePath, err)
		}
	}

	if err := os.MkdirAll(statePath, 0700); err != nil {
		return nil, fmt.Errorf("create state dir [%s]: %w", statePath, err)
	}

	cleanState, err := canonicalizePath(statePath)
	if err != nil {
		return nil, fmt.Errorf("canonicalize state path [%s]: %w", statePath, err)
	}

	// Validate root and state are disjoint after resolving real paths/symlinks
	if err := ValidateDisjoint(cleanRoot, cleanState); err != nil {
		return nil, err
	}

	// Validate same filesystem (Linux st_dev / Windows volume)
	if err := CheckSameFilesystem(cleanRoot, cleanState); err != nil {
		return nil, err
	}

	stagingPath := filepath.Join(cleanState, "staging")
	trashPath := filepath.Join(cleanState, "trash")
	versionPath := filepath.Join(cleanState, "versions")

	for _, dir := range []string{stagingPath, trashPath, versionPath} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("create state subdir %s: %w", dir, err)
		}
	}

	// Anchor os.Root handles
	osRoot, err := os.OpenRoot(cleanRoot)
	if err != nil {
		return nil, fmt.Errorf("open os.Root for root [%s]: %w", cleanRoot, err)
	}

	stateRoot, err := os.OpenRoot(cleanState)
	if err != nil {
		_ = osRoot.Close()
		return nil, fmt.Errorf("open os.Root for state [%s]: %w", cleanState, err)
	}

	svc := &Service{
		storageID:   cfg.StorageID,
		rootPath:    cleanRoot,
		statePath:   cleanState,
		stagingPath: stagingPath,
		trashPath:   trashPath,
		versionPath: versionPath,
		osRoot:      osRoot,
		stateRoot:   stateRoot,
		db:          cfg.DB,
	}

	// Reconcile pending operations; fail initialization if reconciliation fails
	if cfg.DB != nil {
		if err := svc.Reconcile(context.Background()); err != nil {
			_ = svc.Close()
			return nil, fmt.Errorf("startup reconciliation failed: %w", err)
		}
	}

	return svc, nil
}

// Close releases resources associated with the local state service.
func (s *Service) Close() error {
	var errs []error
	if s.osRoot != nil {
		if err := s.osRoot.Close(); err != nil {
			errs = append(errs, err)
		}
		s.osRoot = nil
	}
	if s.stateRoot != nil {
		if err := s.stateRoot.Close(); err != nil {
			errs = append(errs, err)
		}
		s.stateRoot = nil
	}
	return errors.Join(errs...)
}

// RootPath returns the canonical absolute path of the served root.
func (s *Service) RootPath() string {
	return s.rootPath
}

// StatePath returns the canonical absolute path of the state directory.
func (s *Service) StatePath() string {
	return s.statePath
}

// OsRoot returns the anchored os.Root handle for the served root.
func (s *Service) OsRoot() *os.Root {
	return s.osRoot
}

// StateRoot returns the anchored os.Root handle for the state directory.
func (s *Service) StateRoot() *os.Root {
	return s.stateRoot
}

func canonicalizePath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	realPath, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return filepath.Clean(realPath), nil
	}
	// If path does not fully exist yet, canonicalize existing parent
	parent := filepath.Dir(abs)
	realParent, errParent := filepath.EvalSymlinks(parent)
	if errParent == nil {
		return filepath.Clean(filepath.Join(realParent, filepath.Base(abs))), nil
	}
	return filepath.Clean(abs), nil
}

func isEscapingRel(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ValidateDisjoint ensures root and state directories are not identical or nested within each other.
func ValidateDisjoint(root, state string) error {
	cleanRoot := filepath.Clean(root)
	cleanState := filepath.Clean(state)

	if cleanRoot == cleanState {
		return ErrNestedStateRoot
	}

	rel1, err1 := filepath.Rel(cleanRoot, cleanState)
	if err1 == nil && !isEscapingRel(rel1) {
		return fmt.Errorf("%w: state [%s] is inside root [%s]", ErrNestedStateRoot, state, root)
	}

	rel2, err2 := filepath.Rel(cleanState, cleanRoot)
	if err2 == nil && !isEscapingRel(rel2) {
		return fmt.Errorf("%w: root [%s] is inside state [%s]", ErrNestedStateRoot, root, state)
	}

	return nil
}

// ownedDir is a directory opened through an anchored os.Root and, for
// platforms without descriptor-relative rename, its canonical absolute path.
type ownedDir struct {
	file *os.File
	path string
}

// ownedRootForAbsPath resolves an absolute path to the os.Root that owns it,
// the absolute parent directory, the slash-separated relative parent and the
// validated final name. Paths outside both owned roots are rejected.
func (s *Service) ownedRootForAbsPath(absPath string) (*os.Root, string, string, string, error) {
	abs, err := filepath.Abs(absPath)
	if err != nil {
		return nil, "", "", "", err
	}
	cleanAbs := filepath.Clean(abs)

	for _, owned := range []struct {
		base string
		root *os.Root
	}{
		{s.rootPath, s.osRoot},
		{s.statePath, s.stateRoot},
	} {
		rel, relErr := filepath.Rel(owned.base, cleanAbs)
		if relErr != nil || isEscapingRel(rel) {
			continue
		}
		if rel == "." {
			return nil, "", "", "", fmt.Errorf("%w: refusing to rename owned root itself [%s]", ErrPathEscapesRoot, absPath)
		}
		name := filepath.Base(rel)
		if err := ValidateNameComponent(name); err != nil {
			return nil, "", "", "", err
		}
		parentRel := filepath.Dir(rel)
		parentAbs := owned.base
		if parentRel != "." {
			parentAbs = filepath.Join(owned.base, parentRel)
		} else {
			parentRel = ""
		}
		return owned.root, parentAbs, filepath.ToSlash(parentRel), name, nil
	}

	return nil, "", "", "", fmt.Errorf("%w: path [%s] is outside local storage roots", ErrPathEscapesRoot, absPath)
}

// openOwnedDir opens rel (or the root itself when rel is empty) through root.
func openOwnedDir(root *os.Root, absPath, rel string) (ownedDir, error) {
	openRel := rel
	if openRel == "" {
		openRel = "."
	}
	f, err := root.Open(openRel)
	if err != nil {
		return ownedDir{}, fmt.Errorf("open parent directory [%s]: %w", absPath, err)
	}
	return ownedDir{file: f, path: absPath}, nil
}

// ValidateNameComponent ensures a single filename/directory component is safe.
func ValidateNameComponent(name string) error {
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("%w: empty or dot name [%s]", ErrInvalidName, name)
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "\x00") {
		return fmt.Errorf("%w: path separator or NUL in component [%s]", ErrInvalidName, name)
	}
	return nil
}

// CanonicalRelPath converts a storage-relative path (e.g. "/", "/sub/file.txt", "sub/file.txt")
// into a canonical relative path without leading slash (e.g. "", "sub/file.txt").
// Rejects any attempt to use ".." to escape root.
func (s *Service) CanonicalRelPath(storagePath string) (string, error) {
	raw := strings.ReplaceAll(storagePath, "\\", "/")
	parts := strings.Split(raw, "/")
	depth := 0
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			depth--
			if depth < 0 {
				return "", fmt.Errorf("%w: path traverses above root [%s]", ErrPathEscapesRoot, storagePath)
			}
			continue
		}
		depth++
		if err := ValidateNameComponent(part); err != nil {
			return "", err
		}
	}

	cleaned := stdpath.Clean("/" + raw)
	trimmed := strings.TrimPrefix(cleaned, "/")
	if trimmed == "" || trimmed == "." {
		return "", nil // root directory
	}
	return trimmed, nil
}

// DiskToRelative converts an absolute disk path inside rootPath into a canonical relative path.
func (s *Service) DiskToRelative(diskAbsPath string) (string, error) {
	abs, err := filepath.Abs(diskAbsPath)
	if err != nil {
		return "", err
	}
	cleanAbs := filepath.Clean(abs)
	rel, err := filepath.Rel(s.rootPath, cleanAbs)
	if err != nil || isEscapingRel(rel) {
		return "", fmt.Errorf("%w: path [%s] does not belong to root [%s]", ErrPathEscapesRoot, diskAbsPath, s.rootPath)
	}
	if rel == "." || rel == "" {
		return "", nil
	}
	return filepath.ToSlash(rel), nil
}

// RelToDiskPath converts a canonical relative path into an absolute disk path.
func (s *Service) RelToDiskPath(canonicalRel string) (string, error) {
	if canonicalRel == "" || canonicalRel == "." {
		return s.rootPath, nil
	}
	diskPath := filepath.Join(s.rootPath, filepath.FromSlash(canonicalRel))
	rel, err := filepath.Rel(s.rootPath, diskPath)
	if err != nil || isEscapingRel(rel) {
		return "", ErrPathEscapesRoot
	}
	return diskPath, nil
}

// CheckSymlink checks that no component of canonicalRel inside osRoot is a symlink.
func (s *Service) CheckSymlink(canonicalRel string) error {
	if canonicalRel == "" || canonicalRel == "." {
		return nil
	}
	parts := strings.Split(canonicalRel, "/")
	cur := ""
	for _, part := range parts {
		if cur == "" {
			cur = part
		} else {
			cur = cur + "/" + part
		}
		fi, err := s.osRoot.Lstat(cur)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			return err
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: component [%s] is a symlink", ErrSymlinkForbidden, cur)
		}
	}
	return nil
}

// CrossRootRename moves a file between the served root and the state root
// without replacing an existing destination. Both paths must belong to one of
// the two owned roots; each parent directory is opened through that root so the
// rename stays confined to it.
func (s *Service) CrossRootRename(srcAbsPath, dstAbsPath string) error {
	return s.crossRootRename(srcAbsPath, dstAbsPath, false)
}

// CrossRootRenameReplace is CrossRootRename with intentional replacement of an
// existing destination. Only put-style writes (which snapshot the previous
// content first) may use it.
func (s *Service) CrossRootRenameReplace(srcAbsPath, dstAbsPath string) error {
	return s.crossRootRename(srcAbsPath, dstAbsPath, true)
}

func (s *Service) crossRootRename(srcAbsPath, dstAbsPath string, replace bool) error {
	srcRoot, srcParentAbs, srcParentRel, srcName, err := s.ownedRootForAbsPath(srcAbsPath)
	if err != nil {
		return err
	}
	dstRoot, dstParentAbs, dstParentRel, dstName, err := s.ownedRootForAbsPath(dstAbsPath)
	if err != nil {
		return err
	}

	srcDir, err := openOwnedDir(srcRoot, srcParentAbs, srcParentRel)
	if err != nil {
		return err
	}
	defer func() { _ = srcDir.file.Close() }()

	dstDir, err := openOwnedDir(dstRoot, dstParentAbs, dstParentRel)
	if err != nil {
		return err
	}
	defer func() { _ = dstDir.file.Close() }()

	return renameBetweenDirs(srcDir, dstDir, srcName, dstName, !replace)
}
