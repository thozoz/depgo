package static

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/public"
	"github.com/gin-gonic/gin"
)

var static fs.FS

func initStatic() {
	if conf.Conf.DistDir == "" {
		dist, err := fs.Sub(public.Public, "dist")
		if err != nil {
			utils.Log.Fatalf("failed to read dist dir")
		}
		static = dist
		return
	}
	static = os.DirFS(conf.Conf.DistDir)
}

func initIndex() {
	indexFile, err := static.Open("index.html")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			utils.Log.Fatalf("index.html not exist, you may forget to put dist of frontend to public/dist")
		}
		utils.Log.Fatalf("failed to read index.html: %v", err)
	}
	defer func() {
		_ = indexFile.Close()
	}()
	index, err := io.ReadAll(indexFile)
	if err != nil {
		utils.Log.Fatalf("failed to read dist/index.html")
	}
	rawHtml := string(index)
	siteConfig := getSiteConfig()
	replaceMap := map[string]string{
		"cdn: undefined":       fmt.Sprintf("cdn: '%s'", siteConfig.Cdn),
		"base_path: undefined": fmt.Sprintf("base_path: '%s'", siteConfig.BasePath),
	}
	for k, v := range replaceMap {
		rawHtml = strings.Replace(rawHtml, k, v, 1)
	}
	conf.SetRawIndexHtml(rawHtml)
	UpdateIndex()
}

// UpdateIndex rebuilds both SPA shells from the current settings and publishes
// them atomically, so a request in flight always reads one coherent document
// instead of observing a half-rewritten one.
func UpdateIndex() {
	favicon := setting.GetStr(conf.Favicon)
	title := setting.GetStr(conf.SiteTitle)
	customizeHead := setting.GetStr(conf.CustomizeHead)
	customizeBody := setting.GetStr(conf.CustomizeBody)
	mainColor := setting.GetStr(conf.MainColor)

	manageHtml := conf.RawIndexHtml()
	replaceMap1 := map[string]string{
		"https://jsd.nn.ci/gh/alist-org/logo@main/logo.svg": favicon,
		"Loading...":            title,
		"main_color: undefined": fmt.Sprintf("main_color: '%s'", mainColor),
	}
	for k, v := range replaceMap1 {
		manageHtml = strings.Replace(manageHtml, k, v, 1)
	}
	conf.SetManageHtml(manageHtml)

	indexHtml := manageHtml
	replaceMap2 := map[string]string{
		"<!-- customize head -->": customizeHead,
		"<!-- customize body -->": customizeBody,
	}
	for k, v := range replaceMap2 {
		indexHtml = strings.Replace(indexHtml, k, v, 1)
	}
	conf.SetIndexHtml(indexHtml)
}

// mountFolder serves one top-level build folder (assets, images, ...) from the
// embedded dist filesystem.
//
// A plain http.FileServer renders an HTML index for a directory request. That
// leaks the exact build artifact names (and therefore the deployment version) to
// anyone, so directories are hidden: the handler answers 404 for every directory
// path instead of listing it.
//
// StripPrefix is required: the route is /<folder>/*filepath while sub is already
// rooted at <folder>, so without it /assets/x.js is looked up as
// assets/assets/x.js, answers 404 text/plain and the browser refuses the module.
func mountFolder(r gin.IRoutes, root fs.FS, folder string) {
	sub, err := fs.Sub(root, folder)
	if err != nil {
		utils.Log.Fatalf("can't find folder: %s", folder)
	}
	fileServer := http.StripPrefix("/"+folder, http.FileServer(http.FS(noDirectoryListing{sub})))
	prefix := fmt.Sprintf("/%s/*filepath", folder)
	r.GET(prefix, gin.WrapH(fileServer))
	r.HEAD(prefix, gin.WrapH(fileServer))
	// The bare folder path is not matched by the wildcard route on every router
	// version, and it is exactly the path that used to render the index. Answer it
	// explicitly so it can never fall through to the SPA catch-all with a 200.
	bare := "/" + folder
	r.GET(bare, gin.WrapH(fileServer))
	r.HEAD(bare, gin.WrapH(fileServer))
}

func Static(r *gin.RouterGroup, noRoute func(handlers ...gin.HandlerFunc)) {
	initStatic()
	initIndex()
	folders := []string{"assets", "images", "streamer", "static"}
	r.Use(func(c *gin.Context) {
		for i := range folders {
			if strings.HasPrefix(c.Request.RequestURI, fmt.Sprintf("/%s/", folders[i])) {
				c.Header("Cache-Control", "public, max-age=15552000")
			}
		}
	})
	for _, folder := range folders {
		mountFolder(r, static, folder)
	}

	noRoute(func(c *gin.Context) {
		if c.Request.Method != "GET" && c.Request.Method != "POST" {
			c.Status(405)
			return
		}
		c.Header("Content-Type", "text/html")
		c.Status(200)
		if strings.HasPrefix(c.Request.URL.Path, "/@manage") {
			_, _ = c.Writer.WriteString(conf.ManageHtml())
		} else {
			_, _ = c.Writer.WriteString(conf.IndexHtml())
		}
		c.Writer.Flush()
		c.Writer.WriteHeaderNow()
	})
}

// ServeSPA serves the primary React SPA index.html
func ServeSPA(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.Status(200)
	_, _ = c.Writer.WriteString(conf.IndexHtml())
	c.Writer.Flush()
	c.Writer.WriteHeaderNow()
}

// noDirectoryListing wraps a filesystem so that directories are invisible to
// http.FileServer. Opening a directory returns fs.ErrNotExist, which the file
// server turns into a 404 instead of the auto-generated HTML index that would
// otherwise enumerate every build artifact in the folder.
type noDirectoryListing struct {
	fsys fs.FS
}

func (n noDirectoryListing) Open(name string) (fs.File, error) {
	file, err := n.fsys.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if info.IsDir() {
		_ = file.Close()
		return nil, fs.ErrNotExist
	}
	return file, nil
}
