package server

import (
	"github.com/alist-org/alist/v3/cmd/flags"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/sign"
	"github.com/alist-org/alist/v3/internal/stream"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/alist-org/alist/v3/server/handles"
	"github.com/alist-org/alist/v3/server/middlewares"
	"github.com/alist-org/alist/v3/server/static"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Init(e *gin.Engine) {
	urlPath := ""
	if conf.URL != nil {
		urlPath = conf.URL.Path
	}
	if !utils.SliceContains([]string{"", "/"}, urlPath) {
		e.GET("/", func(c *gin.Context) {
			c.Redirect(302, urlPath)
		})
	}
	Cors(e)
	e.Use(middlewares.SecurityHeaders())
	e.Use(middlewares.SessionRefresh)
	g := e.Group(urlPath)
	if conf.Conf.Scheme.HttpPort != -1 && conf.Conf.Scheme.HttpsPort != -1 && conf.Conf.Scheme.ForceHttps {
		e.Use(middlewares.ForceHttps)
	}
	g.Any("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	g.GET("/favicon.ico", handles.Favicon)
	g.GET("/robots.txt", handles.Robots)
	common.SecretKey = []byte(conf.Conf.JwtSecret)
	g.Use(middlewares.StoragesLoaded)
	if conf.Conf.MaxConnections > 0 {
		g.Use(middlewares.MaxAllowed(conf.Conf.MaxConnections))
	}
	// Depgo disables WebDAV and S3 protocols in default runtime to ensure local storage safety
	// WebDav(g.Group("/dav"))
	// S3(g.Group("/s3"))

	downloadLimiter := middlewares.DownloadRateLimiter(stream.ClientDownloadLimit)
	signCheck := middlewares.Down(sign.Verify)
	g.GET("/d/*path", signCheck, downloadLimiter, handles.Down)
	g.GET("/p/*path", signCheck, downloadLimiter, handles.Proxy)
	g.HEAD("/d/*path", signCheck, handles.Down)
	g.HEAD("/p/*path", signCheck, handles.Proxy)
	g.GET("/s/:share_id", static.ServeSPA)
	g.GET("/s/:share_id/*path", static.ServeSPA)
	g.GET("/sd/:share_id", downloadLimiter, handles.ShareDown)
	g.GET("/sd/:share_id/*path", downloadLimiter, handles.ShareDown)
	g.HEAD("/sd/:share_id", handles.ShareDown)
	g.HEAD("/sd/:share_id/*path", handles.ShareDown)
	g.GET("/sp/:share_id", downloadLimiter, handles.ShareProxy)
	g.GET("/sp/:share_id/*path", downloadLimiter, handles.ShareProxy)
	g.HEAD("/sp/:share_id", handles.ShareProxy)
	g.HEAD("/sp/:share_id/*path", handles.ShareProxy)

	api := g.Group("/api")
	auth := api.Group("", middlewares.Auth)

	// Depgo exposes only what its own UI uses and what works on a local disk with
	// one administrator. The rest of the AList management API (users, roles,
	// storages and drivers, settings, offline download, search index, labels,
	// tasks, SSO/LDAP/WebAuthn, archive extraction, ...) is not registered: those
	// endpoints either cannot work here or answer with plausible-looking data that
	// has nothing to do with this deployment.
	api.POST("/auth/login", middlewares.NoStore(), handles.Login)
	api.POST("/auth/login/hash", middlewares.NoStore(), handles.LoginHash)
	auth.GET("/me", handles.CurrentUser)
	auth.POST("/me/update", handles.UpdateCurrent)
	auth.GET("/auth/logout", handles.LogOut)
	auth.GET("/me/sessions", handles.ListMySessions)
	auth.POST("/me/sessions/evict", handles.EvictMySession)

	// Share responses carry or validate an access token, so they must never be
	// cached by a proxy or the browser.
	sharePublic := api.Group("/public", middlewares.NoStore())
	sharePublic.GET("/share/info", handles.GetPublicShareInfo)
	sharePublic.POST("/share/auth", handles.AuthPublicShare)
	sharePublic.POST("/share/list", handles.ListPublicShare)
	sharePublic.POST("/share/get", handles.GetPublicShare)

	_fs(auth.Group("/fs", middlewares.AuthNotGuest))
	share := auth.Group("/share", middlewares.AuthNotGuest)
	share.POST("/create", handles.CreateShare)
	share.POST("/update", handles.UpdateShare)
	share.POST("/disable", handles.DisableShare)
	share.POST("/enable", handles.EnableShare)
	share.GET("/list", handles.ListShares)
	share.POST("/delete", handles.DeleteShare)
	if flags.Debug || flags.Dev {
		debug(g.Group("/debug"))
	}
	static.Static(g, func(handlers ...gin.HandlerFunc) {
		e.NoRoute(handlers...)
	})
}

func _fs(g *gin.RouterGroup) {
	g.Any("/list", handles.FsList)
	g.Any("/get", handles.FsGet)
	g.Any("/dirs", handles.FsDirs)
	g.POST("/mkdir", handles.FsMkdir)
	g.POST("/rename", handles.FsRename)
	g.POST("/move", handles.FsMove)
	g.POST("/remove", handles.FsRemove)
	uploadLimiter := middlewares.UploadRateLimiter(stream.ClientUploadLimit)
	g.PUT("/put", middlewares.FsUp, uploadLimiter, handles.FsStream)
	g.PUT("/form", middlewares.FsUp, uploadLimiter, handles.FsForm)

	// Depgo lifecycle and editor endpoints (administrator only).
	adminFs := g.Group("", middlewares.AuthAdmin)
	adminFs.GET("/trash", handles.FsTrashList)
	adminFs.POST("/trash/restore", handles.FsTrashRestore)
	adminFs.POST("/trash/delete", handles.FsTrashDelete)
	adminFs.GET("/versions", handles.FsVersionsList)
	adminFs.POST("/versions/restore", handles.FsVersionsRestore)
	adminFs.GET("/text", handles.FsTextGet)
	adminFs.PUT("/text", handles.FsTextPut)
}

func Cors(r *gin.Engine) {
	origins := conf.Conf.Cors.AllowOrigins
	if len(origins) == 0 {
		// No cross-origin client is configured, so the API is same-origin only.
		// Installing the middleware anyway would answer every foreign Origin
		// with 403; omitting it leaves the browser's normal same-origin policy
		// in charge and keeps the response free of CORS headers.
		return
	}
	config := cors.DefaultConfig()
	config.AllowOrigins = origins
	config.AllowHeaders = conf.Conf.Cors.AllowHeaders
	config.AllowMethods = conf.Conf.Cors.AllowMethods
	r.Use(cors.New(config))
}

func InitS3(e *gin.Engine) {
	Cors(e)
	S3Server(e.Group("/"))
}
