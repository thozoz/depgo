package handles

import (
	"testing"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

func testObjs(n int) []model.Obj {
	objs := make([]model.Obj, n)
	for i := range objs {
		objs[i] = &model.Object{Name: string(rune('a' + i%26))}
	}
	return objs
}

// TestPaginationClampsInvalidPage is the regression test for a remote panic: a
// page of zero or less produced a negative slice start, and FsArchiveList reached
// this function without normalising its input first.
func TestPaginationClampsInvalidPage(t *testing.T) {
	objs := testObjs(10)

	for _, page := range []int{-5, -1, 0} {
		assert.NotPanicsf(t, func() {
			total, got := pagination(objs, &model.PageReq{Page: page, PerPage: 3})
			assert.Equal(t, 10, total)
			assert.Len(t, got, 3, "a non-positive page must behave like page 1")
		}, "page=%d must not panic", page)
	}
}

// TestPaginationClampsPerPage covers zero, negative and oversized page sizes.
func TestPaginationClampsPerPage(t *testing.T) {
	objs := testObjs(10)

	// perPage == 0 falls back to the default rather than returning nothing.
	_, got := pagination(objs, &model.PageReq{Page: 1, PerPage: 0})
	assert.Len(t, got, 10, "per_page=0 must use the default, and 10 < default")

	// A negative per_page keeps its existing "everything" meaning.
	_, got = pagination(objs, &model.PageReq{Page: 1, PerPage: -1})
	assert.Len(t, got, 10)

	// An oversized per_page is capped so a response cannot be made unbounded.
	_, got = pagination(objs, &model.PageReq{Page: 1, PerPage: 1 << 20})
	assert.Len(t, got, 10, "the object list is smaller than the cap")
}

// TestPaginationCapsWindowOnLargeInput proves the cap actually truncates.
func TestPaginationCapsWindowOnLargeInput(t *testing.T) {
	objs := testObjs(MaxPerPage + 50)
	_, got := pagination(objs, &model.PageReq{Page: 1, PerPage: 1 << 20})
	assert.Len(t, got, MaxPerPage, "per_page must be capped at MaxPerPage")
}

// TestPaginationPageBeyondEnd returns an empty page, not a panic.
func TestPaginationPageBeyondEnd(t *testing.T) {
	objs := testObjs(5)
	assert.NotPanics(t, func() {
		total, got := pagination(objs, &model.PageReq{Page: 99, PerPage: 10})
		assert.Equal(t, 5, total)
		assert.Empty(t, got)
	})
}

// TestPaginationNormalSlices checks the ordinary behaviour is unchanged.
func TestPaginationNormalSlices(t *testing.T) {
	objs := testObjs(10)

	_, got := pagination(objs, &model.PageReq{Page: 1, PerPage: 3})
	assert.Len(t, got, 3)

	_, got = pagination(objs, &model.PageReq{Page: 2, PerPage: 3})
	assert.Len(t, got, 3)

	_, got = pagination(objs, &model.PageReq{Page: 4, PerPage: 3})
	assert.Len(t, got, 1, "the last page holds the remainder")

	_, got = pagination(objs, &model.PageReq{Page: 1, PerPage: 10})
	assert.Len(t, got, 10)
}
