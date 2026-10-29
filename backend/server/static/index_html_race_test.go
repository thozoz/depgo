package static

import (
	"sync"
	"testing"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/stretchr/testify/assert"
)

// TestIndexHtmlSnapshotIsAtomic exercises the concurrent read/write pair the
// server performs in practice: the settings handler republishes the shell while
// requests read it. Run with -race, this would flag the plain-field version.
func TestIndexHtmlSnapshotIsAtomic(t *testing.T) {
	conf.SetIndexHtml("initial")

	var wg sync.WaitGroup
	const iterations = 500

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			conf.SetIndexHtml("builder-updated")
			conf.SetManageHtml("manage-updated")
			conf.SetRawIndexHtml("raw-updated")
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			if got := conf.IndexHtml(); got == "" {
				t.Error("IndexHtml returned an empty snapshot")
				return
			}
			_ = conf.ManageHtml()
			_ = conf.RawIndexHtml()
		}
	}()
	wg.Wait()

	assert.Equal(t, "builder-updated", conf.IndexHtml())
}

// TestIndexHtmlSnapshotPublishesCompleteDocument proves a reader never observes a
// partially written value: each published snapshot equals one of the complete
// documents a writer stored, never a mixture.
func TestIndexHtmlSnapshotPublishesCompleteDocument(t *testing.T) {
	documents := []string{
		"<html><body>one</body></html>",
		"<html><body>two</body></html>",
		"<html><body>three</body></html>",
	}
	conf.SetIndexHtml(documents[0])

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			for _, doc := range documents {
				conf.SetIndexHtml(doc)
			}
		}
	}()

	valid := make(map[string]struct{}, len(documents))
	for _, doc := range documents {
		valid[doc] = struct{}{}
	}
	for i := 0; i < 10000; i++ {
		got := conf.IndexHtml()
		if _, ok := valid[got]; !ok {
			close(stop)
			wg.Wait()
			t.Fatalf("reader observed a document that was never published: %q", got)
		}
	}
	close(stop)
	wg.Wait()
}
