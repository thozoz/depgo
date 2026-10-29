package model

import (
	"testing"
	"time"

	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/stretchr/testify/assert"
)

type fakeObj struct {
	name string
}

func (f fakeObj) GetName() string         { return f.name }
func (f fakeObj) GetSize() int64          { return 0 }
func (f fakeObj) ModTime() time.Time      { return time.Time{} }
func (f fakeObj) CreateTime() time.Time   { return time.Time{} }
func (f fakeObj) IsDir() bool             { return false }
func (f fakeObj) GetHash() utils.HashInfo { return utils.HashInfo{} }
func (f fakeObj) GetID() string           { return f.name }
func (f fakeObj) GetPath() string         { return "/" + f.name }

// TestObjMergeHideFiltersMatchingNames covers the normal case: hidden names are
// dropped from the listing.
func TestObjMergeHideFiltersMatchingNames(t *testing.T) {
	om := NewObjMerge()
	om.InitHideReg(`^secret`)
	got := om.Merge([]Obj{fakeObj{name: "secret.txt"}, fakeObj{name: "public.txt"}})
	assert.Len(t, got, 1)
	assert.Equal(t, "public.txt", got[0].GetName())
}

// TestObjMergeHideInvalidPatternFailsClosed is the regression test for a panic:
// InitHideReg used regexp2.MustCompile on stored expressions, so one malformed
// line crashed every directory listing that evaluated it. An unparsable rule must
// keep the files hidden rather than expose them.
func TestObjMergeHideInvalidPatternFailsClosed(t *testing.T) {
	om := NewObjMerge()
	// An unclosed group is invalid for regexp2.
	om.InitHideReg(`(unclosed`)
	got := om.Merge([]Obj{fakeObj{name: "anything.txt"}})
	assert.Empty(t, got, "an unparsable hide pattern must hide everything, not panic")
}

// TestObjMergeHideSkipsBlankLines makes sure blank lines do not become a
// pattern that matches everything.
func TestObjMergeHideSkipsBlankLines(t *testing.T) {
	om := NewObjMerge()
	om.InitHideReg("^secret\n\n")
	got := om.Merge([]Obj{fakeObj{name: "public.txt"}})
	assert.Len(t, got, 1, "blank lines must not hide unrelated files")
}

// TestObjMergeWithoutHideKeepsEverything records the default behaviour.
func TestObjMergeWithoutHideKeepsEverything(t *testing.T) {
	om := NewObjMerge()
	got := om.Merge([]Obj{fakeObj{name: "a.txt"}, fakeObj{name: "b.txt"}})
	assert.Len(t, got, 2)
}
