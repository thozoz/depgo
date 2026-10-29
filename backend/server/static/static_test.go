package static

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

// buildTestFS mirrors the layout the deployment ships: one directory with the
// hashed build artifacts and a nested directory that must stay hidden too.
func buildTestFS() fstest.MapFS {
	return fstest.MapFS{
		"assets/index-abc.js":     {Data: []byte("console.log(1)")},
		"assets/index-def.css":    {Data: []byte("body{}")},
		"assets/nested/secret.js": {Data: []byte("console.log(2)")},
		"images/logo.svg":         {Data: []byte("<svg/>")},
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	sub, err := fs.Sub(buildTestFS(), ".")
	if err != nil {
		t.Fatalf("sub fs: %v", err)
	}
	return http.FileServer(http.FS(noDirectoryListing{sub}))
}

func TestDirectoryListingIsDisabled(t *testing.T) {
	handler := newTestHandler(t)
	for _, path := range []string{"/", "/assets/", "/assets", "/images/"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("GET %s = %d, want 404 (no directory index)", path, rec.Code)
			}
			if body := rec.Body.String(); contains(body, "<a href=") {
				t.Fatalf("GET %s leaked a directory index: %q", path, body)
			}
		})
	}
}

func TestFilesInsideDirectoriesAreStillServed(t *testing.T) {
	handler := newTestHandler(t)
	cases := map[string]string{
		"/assets/index-abc.js":     "console.log(1)",
		"/assets/index-def.css":    "body{}",
		"/assets/nested/secret.js": "console.log(2)",
		"/images/logo.svg":         "<svg/>",
	}
	for path, want := range cases {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want 200", path, rec.Code)
			}
			if got := rec.Body.String(); got != want {
				t.Fatalf("GET %s body = %q, want %q", path, got, want)
			}
		})
	}
}

func TestMissingFileIsNotFound(t *testing.T) {
	handler := newTestHandler(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/nope.js", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestNoDirectoryListingOpenRejectsDirectories(t *testing.T) {
	fsys := noDirectoryListing{buildTestFS()}
	if _, err := fsys.Open("assets"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("opening a directory should report fs.ErrNotExist, got %v", err)
	}
	file, err := fsys.Open("assets/index-abc.js")
	if err != nil {
		t.Fatalf("opening a file failed: %v", err)
	}
	_ = file.Close()
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestMountFolderServesThroughTheRealRoute is the regression test for the
// production outage where every /assets/*.js answered 404 text/plain. The other
// tests build the handler by hand without the route prefix, which is exactly why
// they could not see it.
func TestMountFolderServesThroughTheRealRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	for _, folder := range []string{"assets", "images"} {
		mountFolder(engine.Group("/"), buildTestFS(), folder)
	}

	for path, want := range map[string]string{
		"/assets/index-abc.js":     "console.log(1)",
		"/assets/nested/secret.js": "console.log(2)",
		"/images/logo.svg":         "<svg/>",
	} {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusOK || rec.Body.String() != want {
			t.Fatalf("GET %s = %d %q, want 200 %q", path, rec.Code, rec.Body.String(), want)
		}
	}

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/index-abc.js", nil))
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") && !strings.HasPrefix(ct, "application/javascript") {
		t.Fatalf("Content-Type = %q, want a javascript type", ct)
	}

	for _, path := range []string{"/assets", "/assets/", "/images/"} {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("GET %s = %d, want 404", path, rec.Code)
		}
	}
}
