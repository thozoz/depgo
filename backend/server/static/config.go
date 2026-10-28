package static

import (
	"strings"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/pkg/utils"
)

type SiteConfig struct {
	BasePath string
	Cdn      string
}

func getSiteConfig() SiteConfig {
	basePath := ""
	if conf.URL != nil {
		basePath = conf.URL.Path
	}
	cdn := ""
	if conf.Conf != nil {
		cdn = strings.ReplaceAll(strings.TrimSuffix(conf.Conf.Cdn, "/"), "$version", conf.WebVersion)
	}
	siteConfig := SiteConfig{
		BasePath: basePath,
		Cdn:      cdn,
	}
	if siteConfig.BasePath != "" {
		siteConfig.BasePath = utils.FixAndCleanPath(siteConfig.BasePath)
	}
	if siteConfig.Cdn == "" {
		siteConfig.Cdn = strings.TrimSuffix(siteConfig.BasePath, "/")
	}
	return siteConfig
}
