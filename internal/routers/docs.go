package routers

import (
	"net/http"
	"strings"

	docs "github.com/emiliogozo/panahon-api-go/internal/docs/public"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func redirectTo(url string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Redirect(http.StatusFound, url)
	}
}

func wrapDocsIndex(handler gin.HandlerFunc, indexURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Param("any") {
		case "", "/":
			c.Redirect(http.StatusFound, indexURL)
		default:
			handler(c)
		}
	}
}

func registerDocsRoutes(api *gin.RouterGroup, swagBasePath string) {
	base := strings.TrimRight(swagBasePath, "/")
	docsIndex := base + "/docs/index.html"
	publicDocsIndex := base + "/public-docs/index.html"

	fullDocs := ginSwagger.WrapHandler(swaggerFiles.Handler)
	api.GET("/docs", redirectTo(docsIndex))
	api.GET("/docs/*any", wrapDocsIndex(fullDocs, docsIndex))

	publicDocs := ginSwagger.CustomWrapHandler(&ginSwagger.Config{
		URL:          "../public-openapi.json",
		Title:        "Panahon Public API",
		DocExpansion: "list",
		DeepLinking:  true,
	}, swaggerFiles.NewHandler())
	api.GET("/public-docs", redirectTo(publicDocsIndex))
	api.GET("/public-docs/*any", wrapDocsIndex(publicDocs, publicDocsIndex))

	api.GET("/public-openapi.json", func(c *gin.Context) {
		spec, err := docs.PublicSpec()
		if err != nil {
			_ = c.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot build public API docs"})
			return
		}

		c.Header("Cache-Control", "public, max-age=300")
		c.Data(http.StatusOK, "application/json; charset=utf-8", spec)
	})
}
